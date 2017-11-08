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

type proxyDeploy interface {
	Deploy(int64, grpc.DialOption, Archive, ...Peer)
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
	Leader() raft.ServerAddress
	Apply([]byte, time.Duration) raft.ApplyFuture
}

type proxyRaft struct {
}

func (t proxyRaft) State() raft.RaftState {
	return raft.Shutdown
}

func (t proxyRaft) Leader() raft.ServerAddress {
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

func (t *proxyQuorum) close() {
	t.client.Close()
}

// QuorumOption option for the quorum rpc.
type QuorumOption func(*Quorum)

// QuorumOptionUpload set the upload storage protocol.
func QuorumOptionUpload(proto uploads.Protocol) QuorumOption {
	return func(q *Quorum) {
		q.UploadProtocol = proto
	}
}

// QuorumOptionCredentials set the dial credentials options for the agent.
// when the credentials are nil then insecure connections are used.
func QuorumOptionCredentials(c credentials.TransportCredentials) QuorumOption {
	return func(q *Quorum) {
		if c == nil {
			q.creds = grpc.WithInsecure()
		} else {
			q.creds = grpc.WithTransportCredentials(c)
		}
	}
}

// NewQuorum ...
func NewQuorum(q *QuorumFSM, c cluster, deploy proxyDeploy, options ...QuorumOption) *Quorum {
	r := &Quorum{
		Events:  make(chan raft.Observation, 200),
		m:       &sync.Mutex{},
		lq:      q,
		proxy:   proxyRaft{},
		pdeploy: deploy,
		creds:   grpc.WithInsecure(),
		c:       c,
		pq:      proxyQuorum{},
		UploadProtocol: uploads.ProtocolFunc(
			func(uid []byte, _ uint64) (uploads.Uploader, error) {
				return uploads.NewTempFileUploader()
			},
		),
	}

	for _, opt := range options {
		opt(r)
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
	creds          grpc.DialOption
	proxy          raftproxy
	lq             *QuorumFSM
	pq             proxyQuorum
	pdeploy        proxyDeploy
}

// Observer - implements raftprotocol observer.
func (t *Quorum) observe() {
	for o := range t.Events {
		switch o.Data.(type) {
		case raft.LeaderObservation:
			t.pq.close()

			if o.Raft.Leader() == "" {
				debugx.Println("leader lost disabling quorum locally")
				t.proxy = proxyRaft{}
				t.pq = proxyQuorum{}
				continue
			}

			peader := t.findLeader(string(o.Raft.Leader()))
			debugx.Println("leader identified, enabling quorum locally", peader.Name, peader.Ip)
			t.proxy = o.Raft
			t.pq = proxyQuorum{
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
func (t Quorum) Deploy(ctx context.Context, req *ProxyDeployRequest) (_ *ProxyDeployResult, err error) {
	var (
		_zero ProxyDeployResult
	)
	t.m.Lock()
	p := t.proxy
	t.m.Unlock()

	debugx.Println("deploy invoked")
	defer debugx.Println("deploy completed")

	switch s := p.State(); s {
	case raft.Leader:
		debugx.Println("proxy deploy initiated")
		t.pdeploy.Deploy(req.Concurrency, t.creds, *req.Archive, PtrToPeers(req.Peers...)...)
		debugx.Println("proxy deploy completed")
		return &_zero, err
	default:
		var (
			c Client
		)

		debugx.Println("forwarding deploy request to leader")
		if c, err = Dial(RPCAddress(t.findLeader(string(p.Leader()))), t.creds); err != nil {
			return &_zero, err
		}

		defer c.Close()
		return &_zero, c.RemoteDeploy(req.Concurrency, *req.Archive, PtrToPeers(req.Peers...)...)
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
	defer debugx.Println("upload completed")

	switch s := p.State(); s {
	case raft.Leader, raft.Follower, raft.Candidate:
	default:
		log.Println("failed upload not a member of quorum", s.String(), p.Leader())
		return errors.Errorf("upload must be run on a member of quorum: %s", s)
	}

	debugx.Println("upload: generating deployment ID")
	if deploymentID, err = bw.GenerateID(); err != nil {
		return err
	}

	debugx.Println("upload: receiving metadata")
	if chunk, err = stream.Recv(); err != nil {
		return errors.WithStack(err)
	}

	debugx.Printf("upload: initializing protocol: %T\n", t.UploadProtocol)
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

		debugx.Println("upload: chunk received")
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
	defer debugx.Println("watch completed")

	switch s := p.State(); s {
	case raft.Leader, raft.Follower, raft.Candidate:
	default:
		return errors.Errorf("watch must be run on a member of quorum: %s", s)
	}

	ctx, done := context.WithCancel(context.Background())

	debugx.Println("event observer: registering")
	o := t.lq.Register(pbObserver{dst: out, done: done})
	debugx.Println("event observer: registered")
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
	debugx.Println("dispatch initiated")
	defer debugx.Println("dispatch completed")
	t.m.Lock()
	p := t.proxy
	dispatch := t.pq.Dispatch
	t.m.Unlock()

	leader := p.Leader()

	if leader == "" {
		return errors.New("unable to dispatch message, no leader currently elected")
	}

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
