package agent

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
	"google.golang.org/grpc/credentials"

	bw "bitbucket.org/jatone/bearded-wookie"
	clusterx "bitbucket.org/jatone/bearded-wookie/cluster"
	"bitbucket.org/jatone/bearded-wookie/deployment/agent"
	"bitbucket.org/jatone/bearded-wookie/uploads"
	"github.com/hashicorp/raft"
	"github.com/pkg/errors"
)

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
	for _, m := range messages {
		if err = t.dst.Send(&m); err != nil {
			t.done()
			return errors.WithStack(err)
		}
	}

	return nil
}

type raftproxy interface {
	State() raft.RaftState
	Leader() string
	Apply([]byte, time.Duration) raft.ApplyFuture
}

type proxyRaft struct {
}

func (t proxyRaft) State() raft.RaftState {
	return raft.Shutdown
}

func (t proxyRaft) Leader() string {
	return ""
}

func (t proxyRaft) Apply(cmd []byte, d time.Duration) raft.ApplyFuture {
	return errorFuture{err: errors.New("apply cannot be executed on a proxy node")}
}

type proxyQuorum struct {
	dialOptions []grpc.DialOption
	peader      agent.Peer
	client      Client
	o           *sync.Once
}

func (t *proxyQuorum) Dispatch(m agent.Message) (err error) {
	t.o.Do(func() {
		if t.client, err = DialClient(clusterx.RPCAddress(t.peader), t.dialOptions...); err != nil {
			t.o = &sync.Once{}
		}
	})

	if err != nil {
		return err
	}

	return t.client.Dispatch(m)
}

// NewQuorum ...
func NewQuorum(q *agent.Quorum, c cluster, creds credentials.TransportCredentials) *Quorum {
	ctx, cancel := context.WithCancel(context.Background())
	r := &Quorum{
		Events: make(chan raft.Observation, 200),
		m:      &sync.Mutex{},
		lq:     q,
		ctx:    ctx,
		cancel: cancel,
		proxy:  proxyRaft{},
		c:      c,
		pq: proxyQuorum{
			dialOptions: []grpc.DialOption{
				grpc.WithTransportCredentials(creds),
			},
			o: &sync.Once{},
		},
		UploadProtocol: uploads.ProtocolFunc(
			func(uid []byte, _ uint64) (uploads.Uploader, error) {
				return uploads.NewTempFileUploader()
			},
		),
	}

	go r.observe()

	return r
}

// Quorum implements quorum functionality.
type Quorum struct {
	UploadProtocol uploads.Protocol
	Events         chan raft.Observation
	m              *sync.Mutex
	c              cluster
	proxy          raftproxy
	lq             *agent.Quorum
	pq             proxyQuorum
	ctx            context.Context
	cancel         context.CancelFunc
}

// Observer - implements raftprotocol observer.
func (t *Quorum) observe() {
	for o := range t.Events {
		switch m := o.Data.(type) {
		case raft.LeaderObservation:
			log.Println("------------ Begin Leader observation --------")
			if m.Leader == "" {
				log.Println("leader lost disabling quorum locally")
				t.proxy = proxyRaft{}
				t.pq = proxyQuorum{}
				t.cancel()
				t.ctx, t.cancel = context.WithCancel(context.Background())
				continue
			}

			log.Println("leader identified, enabling quorum locally")
			t.proxy = o.Raft
			t.pq = proxyQuorum{
				peader: t.findLeader(m.Leader),
				o:      &sync.Once{},
			}
			log.Println("------------ End Leader observation --------")
		}
	}
}

// Upload ...
func (t Quorum) Upload(stream agent.Quorum_UploadServer) (err error) {
	var (
		deploymentID []byte
		checksum     hash.Hash
		location     string
		dst          Uploader
		chunk        *agent.ArchiveChunk
	)
	t.m.Lock()
	p := t.proxy
	t.m.Unlock()

	log.Println("upload invoked")
	switch s := p.State(); s {
	case raft.Leader, raft.Follower, raft.Candidate:
	default:
		return errors.Errorf("upload must be run on a member of quorum: %s", s)
	}

	if deploymentID, err = bw.GenerateID(); err != nil {
		return err
	}

	if chunk, err = stream.Recv(); err != nil {
		return errors.WithStack(err)
	}

	if dst, err = t.UploadProtocol.NewUpload(deploymentID, chunk.GetMetadata().Bytes); err != nil {
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
			return stream.SendAndClose(&agent.Archive{
				Peer:         &tmp,
				Location:     location,
				Checksum:     checksum.Sum(nil),
				DeploymentID: deploymentID,
				Ts:           time.Now().UTC().Unix(),
			})
		}

		if err != nil {
			log.Println("error receiving chunk", err)
			return err
		}

		if checksum, err = dst.Upload(bytes.NewBuffer(chunk.Data)); err != nil {
			log.Println("error uploading chunk", err)
			return err
		}
	}
}

// Watch watch for events.
func (t *Quorum) Watch(_ *agent.WatchRequest, out agent.Quorum_WatchServer) (err error) {
	t.m.Lock()
	p := t.proxy
	t.m.Unlock()

	log.Println("watch invoked")
	switch s := p.State(); s {
	case raft.Leader, raft.Follower, raft.Candidate:
	default:
		return errors.Errorf("watch must be run on a member of quorum: %s", s)
	}

	ctx, done := context.WithCancel(t.ctx)
	log.Println("event observer: registering")
	o := t.lq.Register(pbObserver{dst: out, done: done})
	log.Println("event observer: registered")
	defer t.lq.Remove(o)

	<-ctx.Done()

	return nil
}

// Dispatch record deployment events.
func (t *Quorum) Dispatch(in agent.Quorum_DispatchServer) error {
	var (
		err error
		m   *agent.Message
	)

	t.m.Lock()
	p := t.proxy
	t.m.Unlock()

	leader := p.Leader()

	if leader == "" {
		return errors.New("unable to dispatch message, no leader currently elected")
	}

	dispatch := t.pq.Dispatch
	if p.State() == raft.Leader {
		dispatch = func(m agent.Message) error {
			log.Println("------------------------ applying fsm ------------------------")
			return maybeApply(p, 5*time.Second)(agent.MessageToCommand(m)).Error()
		}
	}

	for m, err = in.Recv(); err == nil; m, err = in.Recv() {
		if err = dispatch(*m); err != nil {
			return err
		}
	}

	return nil
}

func (t Quorum) findLeader(leader string) (_zero agent.Peer) {
	var (
		err    error
		peader agent.Peer
	)
	if leader, _, err = net.SplitHostPort(leader); err != nil {
		log.Println("failed to split hostport", err)
		return _zero
	}

	for _, peader = range t.c.Quorum() {
		if clusterx.RaftAddress(peader) == leader {
			return peader
		}
	}

	return _zero
}

func maybeApply(p raftproxy, d time.Duration) func(cmd []byte, err error) raft.ApplyFuture {
	return func(cmd []byte, err error) raft.ApplyFuture {
		return p.Apply(cmd, d)
	}
}