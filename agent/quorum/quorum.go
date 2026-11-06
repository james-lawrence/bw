// Package quorum implements the distributed FSM used to manage deploys.
package quorum

import (
	"bytes"
	"context"
	"fmt"
	"hash"
	"io"
	"log"
	"sync"
	"time"

	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/status"

	"github.com/hashicorp/memberlist"
	"github.com/hashicorp/raft"
	"github.com/james-lawrence/bw/agent"
	"github.com/james-lawrence/bw/agent/dialers"
	"github.com/james-lawrence/bw/clustering/raftutil"
	"github.com/james-lawrence/bw/internal/errorsx"
	"github.com/james-lawrence/bw/internal/langx"
	"github.com/james-lawrence/bw/internal/slogx"
	"github.com/james-lawrence/bw/storage"
	"github.com/pkg/errors"
)

type stateMachine interface {
	State() raft.RaftState
	Leader() *agent.Peer
	Dispatch(context.Context, ...*agent.Message) error
	Deploy(ctx context.Context, c cluster, dialer dialers.Defaults, by string, dopts *agent.DeployOptions, a *agent.Archive, peers ...*agent.Peer) error
}

type cluster interface {
	Members() []*memberlist.Node
	Get([]byte) *memberlist.Node
	GetN(int, []byte) []*memberlist.Node
	Local() *agent.Peer
	Quorum() []*agent.Peer
	Peers() []*agent.Peer
}

// Option option for the quorum rpc.
type Option func(*Quorum)

// OptionDialer set the dialer used to connect to the cluster.
func OptionDialer(d dialers.Defaults) Option {
	return func(q *Quorum) {
		q.dialer = d
	}
}

// OptionModeChange set the callback invoked when the role of the node within the quorum changes.
func OptionModeChange(fn func(agent.PeerMode)) Option {
	return func(q *Quorum) {
		q.modechange = fn
	}
}

// OptionStateMachineDispatch ...
func OptionStateMachineDispatch(d stateMachine) Option {
	return func(q *Quorum) {
		q.sm = d
	}
}

// New new quorum instance based on the options.
func New(cd agent.ConnectableDispatcher, c cluster, codec transcoder, upload storage.UploadProtocol, rp raftutil.Protocol, options ...Option) Quorum {
	deployment := newDeployment(c)
	obs := NewObserver(cd)
	history := NewHistory()
	leadershipTransfer := NewLeadershipTransfer(c, rp)

	wal := NewWAL(
		NewTranscoder(
			deployment,
			codec,
			obs,
			history,
			leadershipTransfer,
		),
	)

	r := langx.Clone(Quorum{
		deployment:            deployment,
		ConnectableDispatcher: cd,
		wal:                   &wal,
		sm:                    &DisabledMachine{},
		uploads:               upload,
		rp:                    rp,
		dialer:                dialers.NewQuorum(c, grpc.WithTransportCredentials(insecure.NewCredentials())),
		m:                     &sync.Mutex{},
		c:                     c,
		lost:                  make(chan struct{}),
		disconnected:          make(chan struct{}),
		history:               history,
		leadershipTransfer:    leadershipTransfer,
	}, options...)

	return r
}

// Quorum implements quorum functionality.
type Quorum struct {
	agent.ConnectableDispatcher
	deployment         *deployment
	wal                *WAL
	sm                 stateMachine
	uploads            storage.UploadProtocol
	m                  *sync.Mutex
	c                  cluster
	dialer             dialers.Defaults
	lost               chan struct{}
	disconnected       chan struct{}
	rp                 raftutil.Protocol
	history            History
	leadershipTransfer *LeadershipTransfer
	modechange         func(agent.PeerMode)
}

func (t *Quorum) mode(m agent.PeerMode) {
	if t.modechange != nil {
		t.modechange(m)
	}
}

// Observe observes a raft cluster and updates the quorum state.
func (t *Quorum) Observe(events chan raft.Observation) {
	monitor := newApplyMonitor(t, func() *agent.Peer {
		if t.proxy().State() != raft.Leader {
			return nil
		}

		return t.c.Local()
	})
	defer func() { errorsx.Log(monitor.Close()) }()

	go t.rp.Overlay(
		t.c,
		raftutil.ProtocolOptionStateMachine(func() raft.FSM {
			return newMonitoredWAL(t.wal, monitor)
		}),
		raftutil.ProtocolOptionObservers(
			t.leadershipTransfer.NewRaftObserver(),
			raft.NewObserver(events, true, func(o *raft.Observation) bool {
				switch d := o.Data.(type) {
				case raft.LeaderObservation, raft.RaftState:
					return true
				case raft.RequestVoteRequest:
					if d.Term > 10 {
						t.rp.ClusterChange.Broadcast()
					}
					return false
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
				t.lostquorum()
			}
		case raft.LeaderObservation:
			log.Println("leadership observation", o.Raft.State())
			switch o.Raft.State() {
			case raft.Leader:
				t.mode(agent.PeerMode_PeerModeLeader)
				t.sm = func() stateMachine {
					sm := NewMachine(
						t.c.Local(),
						o.Raft,
					)

					// background this task so dispatches work.
					go func(ctx context.Context) {
						errorsx.Log(sm.initialize())
						_ = slogx.Verbose(errors.Wrap(
							t.deployment.restartActiveDeploy(ctx, t.dialer, sm),
							"failed to restart an active deploy",
						))
						errorsx.Log(t.deployment.determineLatestDeploy(ctx, t.dialer, sm))
					}(context.Background())

					return sm
				}()
			case raft.Follower, raft.Candidate:
				t.mode(agent.PeerMode_PeerModeFollower)
				t.sm = func() stateMachine { sm := NewProxyMachine(t.c, o.Raft, t.dialer); return &sm }()
			case raft.Shutdown:
				log.Println("shutdown disabling quorum locally")
				t.mode(agent.PeerMode_PeerModePeer)
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
func (t *Quorum) Deploy(ctx context.Context, by string, dopts *agent.DeployOptions, a *agent.Archive, peers ...*agent.Peer) (err error) {
	return t.proxy().Deploy(ctx, t.c, t.dialer, by, dopts, a, peers...)
}

// Upload ...
func (t *Quorum) Upload(stream agent.Quorum_UploadServer) (err error) {
	var (
		checksum hash.Hash
		location string
		dst      agent.Uploader
		chunk    *agent.UploadChunk
	)

	if chunk, err = stream.Recv(); err != nil {
		return errors.WithStack(err)
	}

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
					Commit:       metadata.Vcscommit,
					Checksum:     checksum.Sum(nil),
					DeploymentID: checksum.Sum(nil),
					Ts:           time.Now().UTC().Unix(),
				},
			})
		}

		if err != nil {
			log.Println("error receiving chunk", err)
			return err
		}

		if _, err = dst.Upload(bytes.NewBuffer(chunk.Data)); err != nil {
			log.Println("error uploading chunk", err)
			return err
		}
	}
}

// Watch watch for events.
func (t *Quorum) History(context.Context) (_ []*agent.Message, err error) {
	if _, err = t.quorumOnly(); err != nil {
		return nil, errors.Wrap(err, "history")
	}

	return t.history.Snapshot(), nil

}

// Watch watch for events.
func (t *Quorum) Watch(cursor agent.Cursor, out agent.Quorum_WatchServer) (err error) {
	disconnected := t.disconnected
	lost := t.lost

	if _, err = t.quorumOnly(); err != nil {
		return errors.Wrap(err, "watch")
	}

	ctx, cancel := context.WithCancelCause(out.Context())
	defer cancel(nil)

	go func() {
		select {
		case <-ctx.Done():
		case <-disconnected:
			cancel(errors.New("disconnected"))
		case <-lost:
			cancel(errors.New("quorum membership lost"))
		}
	}()

	seq := t.ConnectableDispatcher.Connect(cursor)
	for m := range seq.Each(ctx) {
		if err = out.Send(m); err != nil {
			cause := errors.Wrap(err, "failed to deliver message")
			errorsx.Log(cause)
			return cause
		}
	}

	cause := errors.WithStack(errorsx.Compact(context.Cause(ctx), seq.Err()))
	errorsx.Log(cause)
	return cause
}

// Dispatch record deployment events.
func (t *Quorum) Dispatch(ctx context.Context, m ...*agent.Message) (err error) {
	if err = t.proxy().Dispatch(ctx, m...); err != nil {
		log.Println(errors.Wrap(err, "unable to dispatch message"))
		t.rp.ClusterChange.Broadcast()
		return err
	}

	return nil
}

func (t *Quorum) proxy() stateMachine {
	t.m.Lock()
	defer t.m.Unlock()
	return t.sm
}

func (t *Quorum) lostquorum() {
	t.m.Lock()
	defer t.m.Unlock()
	close(t.lost)
	t.lost = make(chan struct{})
}

func (t *Quorum) disconnect() {
	t.m.Lock()
	defer t.m.Unlock()
	close(t.disconnected)
	t.disconnected = make(chan struct{})
	t.rp.ClusterChange.Broadcast()
}

func (t *Quorum) quorumOnly() (p stateMachine, err error) {
	p = t.proxy()
	switch state := p.State(); state {
	case raft.Leader, raft.Follower, raft.Candidate:
		return p, nil
	default:
		log.Printf("broadcasting cluster change due invalid quorum to tickle raft overlay: %T - %s\n", p, state)
		t.disconnect()
		return p, status.Error(codes.Unavailable, fmt.Sprintf("must be run on a member of quorum: %s", state))
	}
}
