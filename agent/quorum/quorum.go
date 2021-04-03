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
	"github.com/james-lawrence/bw/agent"
	"github.com/james-lawrence/bw/agent/dialers"
	"github.com/james-lawrence/bw/clustering/raftutil"
	"github.com/james-lawrence/bw/internal/x/debugx"
	"github.com/james-lawrence/bw/internal/x/logx"
	"github.com/james-lawrence/bw/storage"
	"github.com/pkg/errors"
)

type stateMachine interface {
	State() raft.RaftState
	Leader() *agent.Peer
	Dispatch(context.Context, ...*agent.Message) error
}

type cluster interface {
	LocalNode() *memberlist.Node
	Members() []*memberlist.Node
	Get([]byte) *memberlist.Node
	GetN(int, []byte) []*memberlist.Node
	Local() *agent.Peer
	Quorum() []*agent.Peer
	Peers() []*agent.Peer
}

type deployer interface {
	Deploy(dialers.Defaults, agent.DeployOptions, agent.Archive, ...*agent.Peer) error
}

// Option option for the quorum rpc.
type Option func(*Quorum)

// OptionDialer set the dialer used to connect to the cluster.
func OptionDialer(d dialers.Defaults) Option {
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

// OptionInitializers for the state machine.
func OptionInitializers(inits ...Initializer) Option {
	return func(q *Quorum) {
		q.initializers = inits
	}
}

// New new quorum instance based on the options.
func New(cd agent.ConnectableDispatcher, c cluster, d deployer, codec transcoder, upload storage.UploadProtocol, rp raftutil.Protocol, options ...Option) Quorum {
	deployment := newDeployment(d, c)
	obs := NewObserver(make(chan *agent.Message, 100))
	go obs.Observe(context.Background(), cd)
	wal := NewWAL(
		NewTranscoder(
			deployment,
			codec,
			obs,
		),
	)

	r := Quorum{
		deployment:            deployment,
		ConnectableDispatcher: cd,
		wal:                   &wal,
		sm:                    &DisabledMachine{},
		uploads:               upload,
		rp:                    rp,
		dialer:                dialers.NewQuorum(c, grpc.WithInsecure()),
		m:                     &sync.Mutex{},
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
	deployment   *deployment
	wal          *WAL
	sm           stateMachine
	uploads      storage.UploadProtocol
	m            *sync.Mutex
	c            cluster
	dialer       dialers.Defaults
	lost         chan struct{}
	initializers []Initializer
	rp           raftutil.Protocol
}

// Observe observes a raft cluster and updates the quorum state.
func (t *Quorum) Observe(events chan raft.Observation) {
	go t.rp.Overlay(
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
					sm := NewMachine(
						t.c.Local(),
						o.Raft,
						t.initializers...,
					)

					// background this task so dispatches work.
					go func() {
						logx.MaybeLog(sm.initialize())
						logx.Verbose(errors.Wrap(
							t.deployment.restartActiveDeploy(context.Background(), t.dialer, sm),
							"failed to restart an active deploy",
						))
						logx.MaybeLog(t.deployment.determineLatestDeploy(context.Background(), t.dialer, sm))
					}()

					return sm
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
	return t.deployment.getInfo(t.sm.Leader()), nil
}

// Cancel any active deploys
func (t *Quorum) Cancel(ctx context.Context, req *agent.CancelRequest) (err error) {
	return t.deployment.cancel(ctx, req, t.dialer, t.proxy())
}

// Deploy ...
func (t *Quorum) Deploy(dopts agent.DeployOptions, a agent.Archive, peers ...*agent.Peer) (err error) {
	return logx.MaybeLog(t.deployment.deploy(t.dialer, dopts, a, peers...))
}

// Upload ...
func (t *Quorum) Upload(stream agent.Quorum_UploadServer) (err error) {
	var (
		checksum hash.Hash
		location string
		dst      agent.Uploader
		chunk    *agent.UploadChunk
	)

	debugx.Println("upload invoked")
	defer debugx.Println("upload completed")

	debugx.Println("upload: receiving metadata")
	if chunk, err = stream.Recv(); err != nil {
		return errors.WithStack(err)
	}

	debugx.Printf("upload: initializing protocol: %T\n", t.uploads)
	metadata := chunk.GetMetadata()
	if dst, err = t.uploads.NewUpload(metadata.Bytes); err != nil {
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
					Peer:         tmp,
					Location:     location,
					Checksum:     checksum.Sum(nil),
					DeploymentID: checksum.Sum(nil),
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

	if _, err = t.quorumOnly(); err != nil {
		return errors.Wrap(err, "watch")
	}

	events := make(chan agent.Message)
	if l, s, err = t.ConnectableDispatcher.Connect(events); err != nil {
		return logx.MaybeLog(errors.Wrap(err, "failed to connect to dispatcher"))
	}

	defer l.Close()
	defer s.GracefulStop()

	for {
		select {
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
func (t *Quorum) Dispatch(ctx context.Context, m ...*agent.Message) (err error) {
	return logx.MaybeLog(errors.Wrap(t.proxy().Dispatch(ctx, m...), "failed to dispatch"))
}

func (t *Quorum) proxy() stateMachine {
	t.m.Lock()
	defer t.m.Unlock()
	return t.sm
}

func (t *Quorum) quorumOnly() (p stateMachine, err error) {
	p = t.proxy()
	switch state := p.State(); state {
	case raft.Leader, raft.Follower, raft.Candidate:
		return p, nil
	default:
		log.Printf("broadcasting cluster change due invalid quorum to tickle raft overlay: %T - %s\n", p, state)
		t.rp.ClusterChange.Broadcast()
		return p, errors.Errorf("must be run on a member of quorum: %s", state)
	}
}
