package agent

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

	bw "bitbucket.org/jatone/bearded-wookie"
	"bitbucket.org/jatone/bearded-wookie/uploads"
	"bitbucket.org/jatone/bearded-wookie/x/debugx"
	"bitbucket.org/jatone/bearded-wookie/x/grpcx"
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
	dst  Quorum_WatchServer
	done context.CancelFunc
}

func (t pbObserver) Receive(messages ...Message) (err error) {
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
	peader      Peer
	client      Conn
	o           *sync.Once
}

func (t *proxyQuorum) Dispatch(m Message) (err error) {
	t.o.Do(func() {
		if t.client, err = Dial(RPCAddress(t.peader), t.dialOptions...); err != nil {
			t.o = &sync.Once{}
		}
	})

	if err != nil {
		return errors.Wrap(err, "proxy dispatch dial failure")
	}

	return errors.Wrapf(t.client.Dispatch(m), "proxy dispatch: %s - %s", t.peader.Name, t.peader.Ip)
}

// NewQuorum ...
func NewQuorum(q *QuorumFSM, c cluster, creds credentials.TransportCredentials) *Quorum {
	ctx, cancel := context.WithCancel(context.Background())
	r := &Quorum{
		Events: make(chan raft.Observation, 200),
		m:      &sync.Mutex{},
		lq:     q,
		ctx:    ctx,
		cancel: cancel,
		proxy:  proxyRaft{},
		creds:  creds,
		c:      c,
		pq:     proxyQuorum{},
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
	creds          credentials.TransportCredentials
	proxy          raftproxy
	lq             *QuorumFSM
	pq             proxyQuorum
	ctx            context.Context
	cancel         context.CancelFunc
}

// Observer - implements raftprotocol observer.
func (t *Quorum) observe() {
	for o := range t.Events {
		switch m := o.Data.(type) {
		case raft.LeaderObservation:
			if m.Leader == "" {
				debugx.Println("leader lost disabling quorum locally")
				t.proxy = proxyRaft{}
				t.pq = proxyQuorum{}
				t.cancel()
				t.ctx, t.cancel = context.WithCancel(context.Background())
				continue
			}
			peader := t.findLeader(m.Leader)
			debugx.Println("leader identified, enabling quorum locally", peader.Name, peader.Ip)
			t.proxy = o.Raft
			t.pq = proxyQuorum{
				peader: peader,
				o:      &sync.Once{},
				dialOptions: []grpc.DialOption{
					grpc.WithTransportCredentials(t.creds),
				},
			}
		}
	}
}

// Upload ...
func (t Quorum) Upload(stream Quorum_UploadServer) (err error) {
	var (
		deploymentID []byte
		checksum     hash.Hash
		location     string
		dst          Uploader
		chunk        *ArchiveChunk
	)
	t.m.Lock()
	p := t.proxy
	t.m.Unlock()

	debugx.Println("upload invoked")
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
			return stream.SendAndClose(&Archive{
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
func (t *Quorum) Watch(_ *WatchRequest, out Quorum_WatchServer) (err error) {
	t.m.Lock()
	p := t.proxy
	t.m.Unlock()

	debugx.Println("watch invoked")
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
func (t *Quorum) Dispatch(in Quorum_DispatchServer) error {
	var (
		err error
		m   *Message
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
		dispatch = func(m Message) error {
			return maybeApply(p, 5*time.Second)(MessageToCommand(m)).Error()
		}
	}

	for m, err = in.Recv(); err == nil; m, err = in.Recv() {
		if err = dispatch(*m); err != nil {
			return err
		}
	}

	return nil
}

func (t Quorum) findLeader(leader string) (_zero Peer) {
	var (
		peader Peer
	)

	for _, peader = range t.c.Quorum() {
		if RaftAddress(peader) == leader {
			return peader
		}
	}

	log.Println("-------------------- failed to locate leader --------------------")
	return _zero
}

func maybeApply(p raftproxy, d time.Duration) func(cmd []byte, err error) raft.ApplyFuture {
	return func(cmd []byte, err error) raft.ApplyFuture {
		return p.Apply(cmd, d)
	}
}
