package agent

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"io"
	"time"

	"google.golang.org/grpc"

	"bitbucket.org/jatone/bearded-wookie/x/debugx"
	"github.com/pkg/errors"
)

// DialQuorum ...
func DialQuorum(c cluster, options ...grpc.DialOption) (conn Conn, err error) {
	for _, q := range c.Quorum() {
		if conn, err = Dial(RPCAddress(q), options...); err != nil {
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
	if t.conn == nil {
		return nil
	}

	return t.conn.Close()
}

// Upload ...
func (t Conn) Upload(srcbytes uint64, src io.Reader) (info Archive, err error) {
	var (
		stream Quorum_UploadClient
		_info  *Archive
	)

	rpc := NewQuorumClient(t.conn)
	if stream, err = rpc.Upload(context.Background()); err != nil {
		return info, errors.Wrap(err, "failed to create upload stream")
	}

	initialChunk := &ArchiveChunk{
		Checksum: []byte{},
		Data:     []byte{},
		InitialChunkMetadata: &ArchiveChunk_Metadata{
			Metadata: &UploadMetadata{Bytes: srcbytes},
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

// RemoteDeploy deploy using a remote server to coordinate, takes an archive an a list.
// of servers to deploy to.
func (t Conn) RemoteDeploy(concurrency int64, archive Archive, peers ...Peer) (err error) {
	rpc := NewQuorumClient(t.conn)
	req := ProxyDeployRequest{
		Archive: &archive,
		Peers:   PeersToPtr(peers...),
	}

	if _, err = rpc.Deploy(context.Background(), &req); err != nil {
		return errors.WithStack(err)
	}

	return nil
}

// Deploy ...
func (t Conn) Deploy(info Archive) error {
	var (
		err error
	)

	rpc := NewAgentClient(t.conn)
	if _, err = rpc.Deploy(context.Background(), &info); err != nil {
		return errors.Wrap(err, "failed to initiated deploy")
	}

	return nil
}

// Connect ...
func (t Conn) Connect() (d ConnectInfo, err error) {
	var (
		response *ConnectInfo
	)

	rpc := NewAgentClient(t.conn)
	if response, err = rpc.Connect(context.Background(), &ConnectRequest{}); err != nil {
		return d, errors.WithStack(err)
	}

	return ConnectInfo{
		Secret: response.Secret,
		Quorum: response.Quorum,
	}, nil
}

// Info ...
func (t Conn) Info() (_zeroInfo Status, err error) {
	var (
		_zero StatusRequest
		info  *Status
	)
	rpc := NewAgentClient(t.conn)
	if info, err = rpc.Info(context.Background(), &_zero); err != nil {
		return _zeroInfo, errors.Wrap(err, "failed to retrieve info")
	}

	return *info, nil
}

// Watch watch for messages sent to the leader. blocks.
func (t Conn) Watch(out chan<- Message) (err error) {
	var (
		src Quorum_WatchClient
		msg *Message
	)
	debugx.Println("watch started")
	defer debugx.Println("watch finished")
	defer close(out)

	c := NewQuorumClient(t.conn)
	if src, err = c.Watch(context.Background(), &WatchRequest{}); err != nil {
		return errors.WithStack(err)
	}

	for msg, err = src.Recv(); err == nil; msg, err = src.Recv() {
		out <- *msg
	}

	return errors.WithStack(err)
}

// Dispatch messages to the leader.
func (t Conn) Dispatch(messages ...Message) (err error) {
	var (
		dst Quorum_DispatchClient
	)

	c := NewQuorumClient(t.conn)

	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()

	if dst, err = c.Dispatch(ctx); err != nil {
		return errors.WithStack(err)
	}

	for _, m := range messages {
		if err = dst.Send(&m); err != nil {
			return errors.WithStack(err)
		}
	}

	return nil
}

func (t Conn) streamArchive(src io.Reader, stream Quorum_UploadClient) (err error) {
	buf := make([]byte, 0, 1024*1024)
	emit := func(chunk, checksum []byte) error {
		return errors.Wrap(stream.Send(&ArchiveChunk{
			Checksum:             checksum,
			Data:                 chunk,
			InitialChunkMetadata: &ArchiveChunk_None{},
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