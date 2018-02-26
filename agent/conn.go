package agent

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"io"
	"log"

	"google.golang.org/grpc"

	"github.com/davecgh/go-spew/spew"
	"github.com/james-lawrence/bw/x/debugx"
	"github.com/pkg/errors"
)

// DialQuorum ...
func DialQuorum(c cluster, options ...grpc.DialOption) (conn Conn, err error) {
	for _, q := range c.Quorum() {
		if conn, err = Dial(RPCAddress(q), options...); err != nil {
			log.Println("failed to dial", RPCAddress(q), err)
			continue
		}

		log.Println("quorum connection established", RPCAddress(q), spew.Sdump(q))
		return conn, nil
	}

	return conn, errors.New("failed to connect to a quorum node")
}

// AddressProxyDialQuorum connects to a quorum peer using any agent for bootstrapping.
func AddressProxyDialQuorum(proxy string, options ...grpc.DialOption) (conn Conn, err error) {
	var (
		client Client
	)

	if client, err = Dial(proxy, options...); err != nil {
		return conn, err
	}

	return ProxyDialQuorum(client, options...)
}

// ProxyDialQuorum connects to a quorum peer using any agent for bootstrapping.
func ProxyDialQuorum(c Client, options ...grpc.DialOption) (conn Conn, err error) {
	var (
		cinfo ConnectResponse
	)

	if cinfo, err = c.Connect(); err != nil {
		return conn, err
	}

	for _, q := range PtrToPeers(cinfo.Quorum...) {
		if conn, err = Dial(RPCAddress(q), options...); err != nil {
			log.Println("failed to dial", RPCAddress(q), err)
			continue
		}
		return conn, nil
	}

	return conn, errors.New("failed to bootstrap from the provided peer")
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

// Shutdown causes the agent to shut down. generally the agent process
// should be configured to automatically restart.
func (t Conn) Shutdown() (err error) {
	rpc := NewAgentClient(t.conn)

	if _, err = rpc.Shutdown(context.Background(), &ShutdownRequest{}); err != nil {
		return errors.WithStack(err)
	}

	return nil
}

// Upload ...
func (t Conn) Upload(srcbytes uint64, src io.Reader) (info Archive, err error) {
	var (
		stream Quorum_UploadClient
		_info  *UploadResponse
	)

	rpc := NewQuorumClient(t.conn)
	if stream, err = rpc.Upload(context.Background()); err != nil {
		return info, errors.Wrap(err, "failed to create upload stream")
	}

	initialChunk := &UploadChunk{
		Checksum: []byte{},
		Data:     []byte{},
		InitialChunkMetadata: &UploadChunk_Metadata{
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

	if !bytes.Equal(_info.Archive.Checksum, checksum.Sum(nil)) {
		return info, errors.Errorf("checksums mismatch: archive(%s), expected(%s)", hex.EncodeToString(_info.Archive.Checksum), hex.EncodeToString(checksum.Sum(nil)))
	}

	return *_info.Archive, err
}

// RemoteDeploy deploy using a remote server to coordinate, takes an archive an a list.
// of servers to deploy to.
func (t Conn) RemoteDeploy(dopts DeployOptions, a Archive, peers ...Peer) (err error) {
	rpc := NewQuorumClient(t.conn)
	req := DeployCommandRequest{
		Archive: &a,
		Options: &dopts,
		Peers:   PeersToPtr(peers...),
	}

	if _, err = rpc.Deploy(context.Background(), &req); err != nil {
		return errors.WithStack(err)
	}

	return nil
}

// Deploy ...
func (t Conn) Deploy(options DeployOptions, archive Archive) (d Deploy, err error) {
	var (
		ar *DeployResponse
	)

	rpc := NewAgentClient(t.conn)

	if ar, err = rpc.Deploy(context.Background(), &DeployRequest{Options: &options, Archive: &archive}); err != nil {
		return d, errors.Wrap(err, "failed to initiated deploy")
	}

	if ar.Deploy == nil {
		return d, errors.New("deploy result is nil")
	}

	return *ar.Deploy, nil
}

// Connect ...
func (t Conn) Connect() (d ConnectResponse, err error) {
	var (
		response *ConnectResponse
	)

	rpc := NewAgentClient(t.conn)
	if response, err = rpc.Connect(context.Background(), &ConnectRequest{}); err != nil {
		return d, errors.WithStack(err)
	}

	return *response, nil
}

// Info ...
func (t Conn) Info() (_zeroInfo StatusResponse, err error) {
	var (
		_zero StatusRequest
		info  *StatusResponse
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

	ctx := context.Background()
	if dst, err = c.Dispatch(ctx); err != nil {
		return errors.WithStack(err)
	}

	for _, m := range messages {
		if err = errors.WithStack(dst.Send(&m)); err != nil {
			log.Println("failed to send message", err)
			goto done
		}
	}

done:
	return errors.WithStack(dst.CloseSend())
}

func (t Conn) streamArchive(src io.Reader, stream Quorum_UploadClient) (err error) {
	buf := make([]byte, 0, 1024*1024)
	emit := func(chunk, checksum []byte) error {
		return errors.Wrap(stream.Send(&UploadChunk{
			Checksum:             checksum,
			Data:                 chunk,
			InitialChunkMetadata: &UploadChunk_None{},
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
