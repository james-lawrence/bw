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
	"net"
	"sync"
	"time"

	"google.golang.org/grpc"

	"github.com/hashicorp/memberlist"
	"github.com/hashicorp/raft"
	"github.com/james-lawrence/bw"
	"github.com/james-lawrence/bw/agent"
	"github.com/james-lawrence/bw/agentutil"
	"github.com/james-lawrence/bw/clustering/raftutil"
	"github.com/james-lawrence/bw/storage"
	"github.com/james-lawrence/bw/internal/x/debugx"
	"github.com/james-lawrence/bw/internal/x/logx"
	"github.com/pkg/errors"
)

type stateMachine interface {
	// Info high level information about deployment status.
	Info() (agent.InfoResponse, error)
	// Deploy initiate a deploy.
	Deploy(dopts agent.DeployOptions, a agent.Archive, peers ...agent.Peer) error
	// Cancel current deploy.
	Cancel() error
	// Dispatch a message to the raft cluster.
	Dispatch(context.Context, ...agent.Message) error
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
	Deploy(agent.Dialer, agent.DeployOptions, agent.Archive, ...agent.Peer) error
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
func New(cd agent.ConnectableDispatcher, c cluster, d deployer, upload storage.UploadProtocol, options ...Option) Quorum {
	wal := NewWAL(make(chan agent.Message, 100))
	r := Quorum{
		ConnectableDispatcher: cd,
		wal:                   &wal,
		sm:                    &DisabledMachine{},
		uploads:               upload,
		dialer:                agent.NewDialer(agent.DefaultDialerOptions(grpc.WithInsecure())...),
		m:                     &sync.Mutex{},
		deploy:                d,
		c:                     c,
		lost:                  make(chan struct{}),
	}

	for _, opt := range options {
		opt(&r)
	}

	return r
}

// Quorum implements quorum functionality.
type Quorum struct {
	agent.ConnectableDispatcher
	wal     *WAL
	sm      stateMachine
	uploads storage.UploadProtocol
	m       *sync.Mutex
	c       cluster
	dialer  agent.Dialer
	deploy  deployer
	lost    chan struct{}
}

// Observe observes a raft cluster and updates the quorum state.
func (t *Quorum) Observe(rp raftutil.Protocol, events chan raft.Observation) {
	go func() {
		for m := range t.wal.observer {
			ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
			logx.MaybeLog(errors.Wrap(t.ConnectableDispatcher.Dispatch(ctx, m), "failed to deliver dispatched event to watchers"))
			cancel()
		}
	}()

	go rp.Overlay(
		t.c,
		raftutil.ProtocolOptionStateMachine(func() raft.FSM {
			return t.wal
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
		case raft.RaftState:
			switch o.Raft.State() {
			case raft.Shutdown:
				log.Println("shutting down watchers", o.Raft.State())
				close(t.lost)
				t.lost = make(chan struct{})
			}
		case raft.LeaderObservation:
			log.Println("leadership observation", o.Raft.State())
			switch o.Raft.State() {
			case raft.Leader:
				t.sm = func() stateMachine {
					sm := NewStateMachine(t.wal, t.c, o.Raft, t.dialer, t.deploy)
					logx.MaybeLog(errors.Wrap(sm.restartActiveDeploy(), "failed to restart an active deploy"))
					logx.MaybeLog(sm.determineLatestDeploy(t.c, t.dialer))
					return &sm
				}()
			case raft.Follower, raft.Candidate:
				t.sm = func() stateMachine { sm := NewProxyMachine(t.c, o.Raft, t.dialer); return &sm }()
			case raft.Shutdown:
				log.Println("shutdown disabling quorum locally")
				t.sm = DisabledMachine{}
			}
		}
	}
}

// Info return current info from the leader.
func (t *Quorum) Info(ctx context.Context) (z agent.InfoResponse, err error) {
	debugx.Println("info invoked")
	defer debugx.Println("info completed")

	return t.sm.Info()
}

// Cancel any active deploys
func (t *Quorum) Cancel(ctx context.Context) (err error) {
	if err = agentutil.Cancel(t.c, t.dialer); err != nil {
		return err
	}

	return t.sm.Cancel()
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
	var (
		s *grpc.Server
		l net.Listener
	)

	t.m.Lock()
	p := t.sm
	t.m.Unlock()

	log.Println("watch invoked")
	defer log.Println("watch completed")

	switch state := p.State(); state {
	case raft.Leader, raft.Follower, raft.Candidate:
	default:
		return errors.Errorf("watch must be run on a member of quorum: %s", state)
	}

	events := make(chan agent.Message)
	if l, s, err = t.ConnectableDispatcher.Connect(events); err != nil {
		return logx.MaybeLog(errors.Wrap(err, "failed to connect to dispatcher"))
	}

	defer l.Close()
	defer s.GracefulStop()

	for {
		select {
		// useful code for testing clients for timeouts.
		// case _ = <-time.After(5 * time.Second):
		// 	return logx.MaybeLog(errors.New("timed out"))
		case _ = <-out.Context().Done():
			return logx.MaybeLog(errors.WithStack(out.Context().Err()))
		case _ = <-t.lost:
			return logx.MaybeLog(errors.New("quorum membership lost"))
		case m := <-events:
			if err = out.Send(&m); err != nil {
				return logx.MaybeLog(errors.Wrap(err, "failed to deliver message"))
			}
		}
	}
}

// Dispatch record deployment events.
func (t *Quorum) Dispatch(ctx context.Context, m ...agent.Message) (err error) {
	debugx.Println("dispatch initiated")
	defer debugx.Println("dispatch completed")

	t.m.Lock()
	p := t.sm
	t.m.Unlock()

	return logx.MaybeLog(errors.Wrap(p.Dispatch(ctx, m...), "failed to dispatch"))
}
