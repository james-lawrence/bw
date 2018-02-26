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
	"google.golang.org/grpc/credentials"

	"github.com/hashicorp/memberlist"
	"github.com/hashicorp/raft"
	"github.com/james-lawrence/bw"
	"github.com/james-lawrence/bw/agent"
	"github.com/james-lawrence/bw/clustering/raftutil"
	"github.com/james-lawrence/bw/storage"
	"github.com/james-lawrence/bw/x/debugx"
	"github.com/james-lawrence/bw/x/grpcx"
	"github.com/james-lawrence/bw/x/logx"
	"github.com/pkg/errors"
)

type cluster interface {
	Local() agent.Peer
	Quorum() []agent.Peer
	LocalNode() *memberlist.Node
	Get([]byte) *memberlist.Node
	GetN(int, []byte) []*memberlist.Node
	Peers() []agent.Peer
}

type deploy interface {
	Deploy(grpc.DialOption, agent.Dispatcher, agent.DeployOptions, agent.Archive, ...agent.Peer) error
}

// errorFuture is used to return a static error.
type errorFuture struct {
	err error
}

func (e errorFuture) Error() error {
	return e.err
}

func (e errorFuture) Response() interface{} {
	return nil
}

func (e errorFuture) Index() uint64 {
	return 0
}

type pbObserver struct {
	dst  agent.Quorum_WatchServer
	done context.CancelFunc
}

func (t pbObserver) Receive(messages ...agent.Message) (err error) {
	var (
		cause error
	)

	for _, m := range messages {
		if err = t.dst.Send(&m); err != nil {
			if cause = errors.Cause(err); cause == context.Canceled {
				return nil
			}

			t.done()

			if grpcx.IgnoreShutdownErrors(cause) == nil {
				return nil
			}

			return errors.Wrapf(err, "error type %T", cause)
		}
	}

	return nil
}

// RaftProxy - proxy implementation of the raft protocol
type RaftProxy interface {
	State() raft.RaftState
	Leader() raft.ServerAddress
	Apply([]byte, time.Duration) raft.ApplyFuture
}

// NewRaftProxy utility function for tests.
func NewRaftProxy(leader string, s raft.RaftState, apply func([]byte, time.Duration) raft.ApplyFuture) RaftProxy {
	return proxyRaft{
		leader: leader,
		state:  s,
		apply:  apply,
	}
}

type proxyRaft struct {
	leader string
	state  raft.RaftState
	apply  func(cmd []byte, d time.Duration) raft.ApplyFuture
}

func (t proxyRaft) State() raft.RaftState {
	return t.state
}

func (t proxyRaft) Leader() raft.ServerAddress {
	return raft.ServerAddress(t.leader)
}

func (t proxyRaft) Apply(cmd []byte, d time.Duration) raft.ApplyFuture {
	if t.apply == nil {
		return errorFuture{err: errors.New("apply cannot be executed on a proxy node")}
	}

	return t.apply(cmd, d)
}

type stateMachineDispatch struct {
	sm *StateMachine
}

func (t stateMachineDispatch) Dispatch(m ...agent.Message) error {
	t.sm.Dispatch(m...)
	return nil
}

type disabledDispatch struct{}

func (disabledDispatch) Dispatch(m ...agent.Message) error {
	return errors.New("dispatch disabled not currently enable for this server")
}

type raftDispatch struct {
	p RaftProxy
}

func (t raftDispatch) Dispatch(messages ...agent.Message) (err error) {
	for _, m := range messages {
		var (
			ok     bool
			b      []byte
			future raft.ApplyFuture
		)

		if b, err = MessageToCommand(m); err != nil {
			return err
		}

		future = t.p.Apply(b, 5*time.Second)

		if err = future.Error(); err != nil {
			return err
		}

		if err, ok = future.Response().(error); ok {
			return err
		}
	}

	return nil
}

type proxyDispatch struct {
	dialOptions []grpc.DialOption
	peader      agent.Peer
	client      agent.Conn
	o           *sync.Once
}

func (t *proxyDispatch) Dispatch(messages ...agent.Message) (err error) {
	t.o.Do(func() {
		if t.client, err = agent.Dial(agent.RPCAddress(t.peader), t.dialOptions...); err != nil {
			t.o = &sync.Once{}
		}
	})

	if err != nil {
		return errors.Wrap(err, "proxy dispatch dial failure")
	}

	for _, m := range messages {
		if err = errors.Wrapf(t.client.Dispatch(m), "proxy dispatch: %s - %s", t.peader.Name, t.peader.Ip); err != nil {
			return err
		}
	}

	return nil
}

func (t *proxyDispatch) close() {
	t.client.Close()
}

// Option option for the quorum rpc.
type Option func(*Quorum)

// OptionCredentials set the dial credentials options for the agent.
// when the credentials are nil then insecure connections are used.
func OptionCredentials(c credentials.TransportCredentials) Option {
	return func(q *Quorum) {
		if c == nil {
			q.creds = grpc.WithInsecure()
		} else {
			q.creds = grpc.WithTransportCredentials(c)
		}
	}
}

// OptionRaftProxy ...
func OptionRaftProxy(proxy RaftProxy) Option {
	return func(q *Quorum) {
		q.proxy = proxy
	}
}

// OptionLeaderDispatch ...
func OptionLeaderDispatch(d agent.Dispatcher) Option {
	return func(q *Quorum) {
		q.ldispatch = d
	}
}

// OptionStateMachineDispatch ...
func OptionStateMachineDispatch(d *StateMachine) Option {
	return func(q *Quorum) {
		q.stateMachine = d
	}
}

func disabledRaftProxy() RaftProxy {
	return NewRaftProxy("", raft.Shutdown, nil)
}

// New new quorum instance based on the options.
func New(c cluster, d deploy, upload storage.UploadProtocol, options ...Option) Quorum {
	sm := NewStateMachine()
	r := Quorum{
		stateMachine: sm,
		uploads:      upload,
		creds:        grpc.WithInsecure(),
		m:            &sync.Mutex{},
		proxy:        disabledRaftProxy(),
		pdispatch:    &proxyDispatch{o: &sync.Once{}},
		ldispatch:    stateMachineDispatch{sm: sm},
		deploy:       d,
		c:            c,
	}

	for _, opt := range options {
		opt(&r)
	}

	return r
}

// Quorum implements quorum functionality.
type Quorum struct {
	stateMachine *StateMachine
	uploads      storage.UploadProtocol
	m            *sync.Mutex
	c            cluster
	creds        grpc.DialOption
	proxy        RaftProxy
	pdispatch    *proxyDispatch
	ldispatch    agent.Dispatcher
	deploy       deploy
}

// Observe observes a raft cluster and updates the quorum state.
func (t *Quorum) Observe(rp raftutil.Protocol, events chan raft.Observation) {
	go rp.Overlay(
		t.c,
		raftutil.ProtocolOptionStateMachine(func() raft.FSM {
			return t.stateMachine
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
			t.pdispatch.close()

			if o.Raft.Leader() == "" {
				debugx.Println("leader lost disabling quorum locally")
				t.proxy = disabledRaftProxy()
				t.pdispatch = &proxyDispatch{o: &sync.Once{}}
				t.ldispatch = disabledDispatch{}
				continue
			}

			peader := t.findLeader(string(o.Raft.Leader()))
			debugx.Println("leader identified, enabling quorum locally", peader.Name, peader.Ip)
			t.proxy = o.Raft
			t.ldispatch = raftDispatch{p: o.Raft}
			t.pdispatch = &proxyDispatch{
				peader: peader,
				o:      &sync.Once{},
				dialOptions: []grpc.DialOption{
					t.creds,
				},
			}
		}
	}
}

// Deploy ...
func (t *Quorum) Deploy(dopts agent.DeployOptions, archive agent.Archive, peers ...agent.Peer) (err error) {
	t.m.Lock()
	p := t.proxy
	t.m.Unlock()

	debugx.Println("deploy invoked")
	defer debugx.Println("deploy completed")

	switch s := p.State(); s {
	case raft.Leader:
		debugx.Println("deploy command initiated")
		defer debugx.Println("deploy command completed")
		return logx.MaybeLog(t.deploy.Deploy(t.creds, t.ldispatch, dopts, archive, peers...))
	default:
		var (
			c agent.Client
		)

		debugx.Println("forwarding deploy request to leader")
		if c, err = agent.Dial(agent.RPCAddress(t.findLeader(string(p.Leader()))), t.creds); err != nil {
			return err
		}

		defer c.Close()
		return c.RemoteDeploy(dopts, archive, peers...)
	}
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
	if deploymentID, err = bw.GenerateID(); err != nil {
		return err
	}

	debugx.Println("upload: receiving metadata")
	if chunk, err = stream.Recv(); err != nil {
		return errors.WithStack(err)
	}

	debugx.Printf("upload: initializing protocol: %T\n", t.uploads)
	if dst, err = t.uploads.NewUpload(deploymentID, chunk.GetMetadata().Bytes); err != nil {
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
	p := t.proxy
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
	o := t.stateMachine.Register(pbObserver{dst: out, done: done})
	debugx.Println("event observer: registered")
	defer t.stateMachine.Remove(o)

	<-ctx.Done()

	return nil
}

// Dispatch record deployment events.
func (t *Quorum) Dispatch(in agent.Quorum_DispatchServer) error {
	var (
		err      error
		m        *agent.Message
		dispatch agent.Dispatcher
	)
	debugx.Println("dispatch initiated")
	defer debugx.Println("dispatch completed")
	t.m.Lock()
	p := t.proxy
	t.m.Unlock()

	dispatch = t.pdispatch
	if p.State() == raft.Leader {
		dispatch = t.ldispatch
	}

	for m, err = in.Recv(); err == nil; m, err = in.Recv() {
		if err = dispatch.Dispatch(*m); err != nil {
			return err
		}
	}

	return nil
}

func (t Quorum) findLeader(leader string) (_zero agent.Peer) {
	var (
		peader agent.Peer
	)

	for _, peader = range t.c.Peers() {
		if agent.RaftAddress(peader) == leader {
			return peader
		}
	}

	log.Println("-------------------- failed to locate leader --------------------")
	return _zero
}
