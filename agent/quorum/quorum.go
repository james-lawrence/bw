// Package quorum implements the distributed FSM used to manage deploys.
// TODO:
//  - create a agent connection pool.
package quorum

import (
	"bytes"
	"context"
	"hash"
	"io"
	"log"
	"sync"
	"time"

	"google.golang.org/grpc"

	"github.com/hashicorp/memberlist"
	"github.com/hashicorp/raft"
	"github.com/james-lawrence/bw"
	"github.com/james-lawrence/bw/agent"
	"github.com/james-lawrence/bw/clustering/raftutil"
	"github.com/james-lawrence/bw/storage"
	"github.com/james-lawrence/bw/x/debugx"
	"github.com/james-lawrence/bw/x/logx"
	"github.com/pkg/errors"
)

type stateMachine interface {
	Deploy(dopts agent.DeployOptions, a agent.Archive, peers ...agent.Peer) error
	// Cancel a deploy.
	Cancel() error
	Dispatch(...agent.Message) error
	State() raft.RaftState
}

type cluster interface {
	Local() agent.Peer
	Quorum() []agent.Peer
	LocalNode() *memberlist.Node
	Members() []*memberlist.Node
	Get([]byte) *memberlist.Node
	GetN(int, []byte) []*memberlist.Node
	Peers() []agent.Peer
}

type deployer interface {
	Deploy(agent.Dialer, agent.Dispatcher, agent.DeployOptions, agent.Archive, ...agent.Peer) error
}

// Option option for the quorum rpc.
type Option func(*Quorum)

// OptionDialer set the dialer used to connect to the cluster.
func OptionDialer(d agent.Dialer) Option {
	return func(q *Quorum) {
		q.dialer = d
	}
}

// OptionStateMachineDispatch ...
func OptionStateMachineDispatch(d stateMachine) Option {
	return func(q *Quorum) {
		q.sm = d
	}
}

// New new quorum instance based on the options.
func New(c cluster, d deployer, upload storage.UploadProtocol, options ...Option) Quorum {
	r := Quorum{
		EventBus: agent.NewEventBusDefault(),
		bus:      make(chan agent.Message, 100),
		sm:       &DisabledMachine{},
		uploads:  upload,
		dialer:   agent.NewDialer(grpc.WithInsecure()),
		m:        &sync.Mutex{},
		deploy:   d,
		c:        c,
	}

	for _, opt := range options {
		opt(&r)
	}

	return r
}

// Quorum implements quorum functionality.
type Quorum struct {
	agent.EventBus
	bus     chan agent.Message
	sm      stateMachine
	uploads storage.UploadProtocol
	m       *sync.Mutex
	c       cluster
	dialer  agent.Dialer
	deploy  deployer
}

// Observe observes a raft cluster and updates the quorum state.
func (t *Quorum) Observe(rp raftutil.Protocol, events chan raft.Observation) {
	go func() {
		for m := range t.bus {
			t.EventBus.Dispatch(m)
		}
	}()
	go rp.Overlay(
		t.c,
		raftutil.ProtocolOptionStateMachine(func() raft.FSM {
			wal := NewWAL()
			return &wal
		}),
		raftutil.ProtocolOptionObservers(
			raft.NewObserver(events, true, func(o *raft.Observation) bool {
				switch o.Data.(type) {
				case raft.LeaderObservation, raft.RaftState:
					return true
				default:
					return false
				}
			}),
		),
	)

	for o := range events {
		switch o.Data.(type) {
		case raft.LeaderObservation:
			if o.Raft.Leader() == "" {
				debugx.Println("leader lost disabling quorum locally")
				t.sm = DisabledMachine{}
				continue
			}

			t.sm = func() stateMachine { sm := NewProxyMachine(t.c, o.Raft, t.dialer); return &sm }()
			if o.Raft.State() == raft.Leader {
				t.sm = func() stateMachine {
					sm := NewStateMachine(t.c, o.Raft, t.dialer, t.deploy, STOObserver(t.bus))
					return &sm
				}()
			}
		}
	}
}

// Deploy ...
func (t *Quorum) Deploy(dopts agent.DeployOptions, a agent.Archive, peers ...agent.Peer) (err error) {
	debugx.Println("deploy invoked")
	defer debugx.Println("deploy completed")
	return logx.MaybeLog(t.sm.Deploy(dopts, a, peers...))
}

// Upload ...
func (t *Quorum) Upload(stream agent.Quorum_UploadServer) (err error) {
	var (
		deploymentID []byte
		checksum     hash.Hash
		location     string
		dst          agent.Uploader
		chunk        *agent.UploadChunk
	)

	debugx.Println("upload invoked")
	defer debugx.Println("upload completed")

	debugx.Println("upload: generating deployment ID")
	if deploymentID, err = bw.SimpleGenerateID(); err != nil {
		return err
	}

	debugx.Println("upload: receiving metadata")
	if chunk, err = stream.Recv(); err != nil {
		return errors.WithStack(err)
	}

	debugx.Printf("upload: initializing protocol: %T\n", t.uploads)
	metadata := chunk.GetMetadata()
	if dst, err = t.uploads.NewUpload(deploymentID, metadata.Bytes); err != nil {
		return err
	}

	for {
		chunk, err := stream.Recv()

		if err == io.EOF {
			if checksum, location, err = dst.Info(); err != nil {
				log.Println("error getting archive info", err)
				return err
			}

			tmp := t.c.Local()
			return stream.SendAndClose(&agent.UploadResponse{
				Archive: &agent.Archive{
					Peer:         &tmp,
					Location:     location,
					Checksum:     checksum.Sum(nil),
					DeploymentID: deploymentID,
					Initiator:    metadata.Initiator,
					Ts:           time.Now().UTC().Unix(),
				},
			})
		}

		if err != nil {
			log.Println("error receiving chunk", err)
			return err
		}

		debugx.Println("upload: chunk received")
		if checksum, err = dst.Upload(bytes.NewBuffer(chunk.Data)); err != nil {
			log.Println("error uploading chunk", err)
			return err
		}
	}
}

// Watch watch for events.
func (t *Quorum) Watch(out agent.Quorum_WatchServer) (err error) {
	t.m.Lock()
	p := t.sm
	t.m.Unlock()

	debugx.Println("watch invoked")
	defer debugx.Println("watch completed")

	switch s := p.State(); s {
	case raft.Leader, raft.Follower, raft.Candidate:
	default:
		return errors.Errorf("watch must be run on a member of quorum: %s", s)
	}

	ctx, done := context.WithCancel(context.Background())

	debugx.Println("event observer: registering")
	o := t.Register(pbObserver{dst: out, done: done})
	debugx.Println("event observer: registered")
	defer t.Remove(o)

	<-ctx.Done()

	return nil
}

// Dispatch record deployment events.
func (t *Quorum) Dispatch(in agent.Quorum_DispatchServer) (err error) {
	var (
		m *agent.Message
	)
	debugx.Println("dispatch initiated")
	defer debugx.Println("dispatch completed")

	for m, err = in.Recv(); err == nil; m, err = in.Recv() {
		t.m.Lock()
		p := t.sm
		t.m.Unlock()

		if err = p.Dispatch(*m); err != nil {
			return err
		}
	}

	return nil
}
