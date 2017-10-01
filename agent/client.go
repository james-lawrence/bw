package agent

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"io"

	"google.golang.org/grpc"

	clusterx "bitbucket.org/jatone/bearded-wookie/cluster"
	"bitbucket.org/jatone/bearded-wookie/deployment/agent"
	"bitbucket.org/jatone/bearded-wookie/x/debugx"
	"github.com/pkg/errors"
)

// DialQuorum ...
func DialQuorum(c cluster, options ...grpc.DialOption) (conn Conn, err error) {
	for _, q := range c.Quorum() {
		if conn, err = Dial(clusterx.RPCAddress(q), options...); err != nil {
			continue
		}

		return conn, nil
	}

	return conn, errors.New("failed to connect to a quorum node")
}

// Dial connects to a node at the given address.
func Dial(address string, options ...grpc.DialOption) (_ignored Conn, err error) {
	var (
		conn *grpc.ClientConn
	)

	if conn, err = grpc.Dial(address, options...); err != nil {
		return _ignored, errors.Wrap(err, "failed to connect to peer")
	}

	return Conn{conn: conn}, nil
}

// Conn a connection to the cluster. implements the Client interface.
type Conn struct {
	conn *grpc.ClientConn
}

// Close ...
func (t Conn) Close() error {
	return t.conn.Close()
}

// Upload ...
func (t Conn) Upload(srcbytes uint64, src io.Reader) (info agent.Archive, err error) {
	var (
		stream agent.Quorum_UploadClient
		_info  *agent.Archive
	)

	rpc := agent.NewQuorumClient(t.conn)
	if stream, err = rpc.Upload(context.Background()); err != nil {
		return info, errors.Wrap(err, "failed to create upload stream")
	}

	initialChunk := &agent.ArchiveChunk{
		Checksum: []byte{},
		Data:     []byte{},
		InitialChunkMetadata: &agent.ArchiveChunk_Metadata{
			Metadata: &agent.UploadMetadata{Bytes: srcbytes},
		},
	}

	// send initial empty chunk with metadata.
	if err = stream.Send(initialChunk); err != nil {
		stream.CloseSend()
		return info, err
	}

	checksum := sha256.New()
	if err = t.streamArchive(io.TeeReader(src, checksum), stream); err != nil {
		stream.CloseSend()
		return info, err
	}

	if _info, err = stream.CloseAndRecv(); err != nil {
		return info, errors.Wrap(err, "failed to receive archive")
	}

	if !bytes.Equal(_info.Checksum, checksum.Sum(nil)) {
		return info, errors.Errorf("checksums mismatch: archive(%s), expected(%s)", hex.EncodeToString(_info.Checksum), hex.EncodeToString(checksum.Sum(nil)))
	}

	return *_info, err
}

// Deploy ...
func (t Conn) Deploy(info agent.Archive) error {
	var (
		err error
	)

	rpc := agent.NewAgentClient(t.conn)
	if _, err = rpc.Deploy(context.Background(), &info); err != nil {
		return errors.Wrap(err, "failed to initiated deploy")
	}

	return nil
}

// Connect ...
func (t Conn) Connect() (d agent.ConnectInfo, err error) {
	var (
		response *agent.ConnectInfo
	)

	rpc := agent.NewAgentClient(t.conn)
	if response, err = rpc.Connect(context.Background(), &agent.ConnectRequest{}); err != nil {
		return d, errors.WithStack(err)
	}

	return agent.ConnectInfo{
		Secret: response.Secret,
		Quorum: response.Quorum,
	}, nil
}

// Info ...
func (t Conn) Info() (_zeroInfo agent.Status, err error) {
	var (
		_zero agent.StatusRequest
		info  *agent.Status
	)
	rpc := agent.NewAgentClient(t.conn)
	if info, err = rpc.Info(context.Background(), &_zero); err != nil {
		return _zeroInfo, errors.Wrap(err, "failed to initiated deploy")
	}

	return *info, nil
}

// Watch watch for messages sent to the leader. blocks.
func (t Conn) Watch(out chan<- agent.Message) (err error) {
	var (
		src agent.Quorum_WatchClient
		msg *agent.Message
	)
	debugx.Println("watch started")
	defer debugx.Println("watch finished")

	c := agent.NewQuorumClient(t.conn)
	if src, err = c.Watch(context.Background(), &agent.WatchRequest{}); err != nil {
		return errors.WithStack(err)
	}

	for msg, err = src.Recv(); err == nil; msg, err = src.Recv() {
		out <- *msg
	}

	return errors.WithStack(err)
}

// Dispatch messages to the leader.
func (t Conn) Dispatch(messages ...agent.Message) (err error) {
	var (
		dst agent.Quorum_DispatchClient
	)

	c := agent.NewQuorumClient(t.conn)

	if dst, err = c.Dispatch(context.Background()); err != nil {
		return errors.WithStack(err)
	}

	for _, m := range messages {
		if err = dst.Send(&m); err != nil {
			return errors.WithStack(err)
		}
	}

	return nil
}

func (t Conn) streamArchive(src io.Reader, stream agent.Quorum_UploadClient) (err error) {
	buf := make([]byte, 0, 1024*1024)
	emit := func(chunk, checksum []byte) error {
		return errors.Wrap(stream.Send(&agent.ArchiveChunk{
			Checksum:             checksum,
			Data:                 chunk,
			InitialChunkMetadata: &agent.ArchiveChunk_None{},
		}), "failed to write chunk")
	}

	for {
		buffer := bytes.NewBuffer(buf)
		checksum := sha256.New()

		if _, err = io.CopyN(buffer, io.TeeReader(src, checksum), int64(buffer.Cap())); err == io.EOF {
			return emit(buffer.Bytes(), checksum.Sum(nil))
		} else if err != nil {
			return errors.Wrap(err, "failed to copy chunk")
		}

		if err = emit(buffer.Bytes(), checksum.Sum(nil)); err != nil {
			return err
		}
	}
}
