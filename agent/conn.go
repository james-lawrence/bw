package agent

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"io"

	"google.golang.org/grpc"

	"github.com/james-lawrence/bw/internal/debugx"
	"github.com/james-lawrence/bw/internal/errorsx"
	"github.com/james-lawrence/bw/internal/iox"
	"github.com/pkg/errors"
)

// MaybeConn hack to retrieve the udnerlying grpc.ClientConn until the dial sutation is resolved.
func MaybeConn(c Client, err error) (*grpc.ClientConn, error) {
	if err != nil {
		return nil, err
	}
	if cc := c.Conn(); cc != nil {
		return cc, nil
	}

	return nil, errorsx.String("invalid client, missing connection")
}

// MaybeClient create a client from a grpc client conn.
func MaybeClient(c *grpc.ClientConn, err error) (Client, error) {
	if err != nil {
		return nil, err
	}

	return NewConn(c), nil
}

// NewConn create a new connection.
func NewConn(c *grpc.ClientConn) Conn {
	return Conn{conn: c}
}

// Conn a connection to the cluster. implements the Client interface.
type Conn struct {
	conn *grpc.ClientConn
}

// Conn - return the underlying grpc connection, this is a hack until the dial situation is resolved.
func (t Conn) Conn() *grpc.ClientConn {
	return t.conn
}

// Close ...
func (t Conn) Close() error {
	if t.conn == nil {
		return nil
	}

	return t.conn.Close()
}

// NodeCancel causes the current deploy (if any) to be cancelled.
func (t Conn) NodeCancel(ctx context.Context) (err error) {
	rpc := NewAgentClient(t.conn)
	if _, err = rpc.Cancel(ctx, &CancelRequest{}); err != nil {
		return errors.WithStack(err)
	}

	return nil
}

// Shutdown causes the agent to shut down. generally the agent process
// should be configured to automatically restart.
func (t Conn) Shutdown(ctx context.Context) (err error) {
	rpc := NewAgentClient(t.conn)

	if _, err = rpc.Shutdown(ctx, &ShutdownRequest{}); err != nil {
		return errors.WithStack(err)
	}

	return nil
}

// Cancel proxy the cancellation through the quorum nodes.
// this cleans up the raft state in addition to the individual nodes.
func (t Conn) Cancel(ctx context.Context, req *CancelRequest) error {
	_, err := NewQuorumClient(t.conn).Cancel(ctx, req)
	return errors.WithStack(err)
}

// QuorumInfo returns high level details about the state of the cluster.
func (t Conn) QuorumInfo(ctx context.Context) (z *InfoResponse, err error) {
	var ()

	rpc := NewQuorumClient(t.conn)
	if z, err = rpc.Info(ctx, &InfoRequest{}); err != nil {
		return z, errors.WithStack(err)
	}

	return z, nil
}

// Upload ...
func (t Conn) Upload(ctx context.Context, meta *UploadMetadata, src io.Reader) (info *Archive, err error) {
	var (
		stream Quorum_UploadClient
		_info  *UploadResponse
	)

	rpc := NewQuorumClient(t.conn)
	if stream, err = rpc.Upload(ctx); err != nil {
		return info, errors.Wrap(err, "failed to create upload stream")
	}

	initialChunk := &UploadChunk{
		Checksum: []byte{},
		Data:     []byte{},
		InitialChunkMetadata: &UploadChunk_Metadata{
			Metadata: meta,
		},
	}

	// send initial empty chunk with metadata.
	if err = stream.Send(initialChunk); err != nil {
		return info, errorsx.Compact(err, stream.CloseSend())
	}

	checksum := sha256.New()
	if err = streamArchive(io.TeeReader(src, checksum), stream); err != nil {
		return info, errorsx.Compact(err, stream.CloseSend())
	}

	if _info, err = stream.CloseAndRecv(); err != nil {
		return info, errors.Wrap(err, "failed to receive archive")
	}

	if !bytes.Equal(_info.Archive.Checksum, checksum.Sum(nil)) {
		return info, errors.Errorf("checksums mismatch: archive(%s), expected(%s)", hex.EncodeToString(_info.Archive.Checksum), hex.EncodeToString(checksum.Sum(nil)))
	}

	return _info.Archive, err
}

// RemoteDeploy deploy using a remote server to coordinate, takes an archive an a list.
// of servers to deploy to.
func (t Conn) RemoteDeploy(ctx context.Context, initiator string, dopts *DeployOptions, a *Archive, peers ...*Peer) (err error) {
	rpc := NewQuorumClient(t.conn)
	req := DeployCommandRequest{
		Initiator: initiator,
		Archive:   a,
		Options:   dopts,
		Peers:     peers,
	}

	if _, err = rpc.Deploy(ctx, &req); err != nil {
		return errors.WithStack(err)
	}

	return nil
}

// Deploy ...
func (t Conn) Deploy(ctx context.Context, options *DeployOptions, archive *Archive) (d *Deploy, err error) {
	var (
		ar *DeployResponse
	)

	rpc := NewAgentClient(t.conn)

	if ar, err = rpc.Deploy(ctx, &DeployRequest{Options: options, Archive: archive}); err != nil {
		return d, errors.Wrap(err, "failed to initiated deploy")
	}

	if ar.Deploy == nil {
		return d, errors.New("deploy result is nil")
	}

	return ar.Deploy, nil
}

// Connect ...
func (t Conn) Connect(ctx context.Context) (d *ConnectResponse, err error) {
	rpc := NewAgentClient(t.conn)
	if d, err = rpc.Connect(ctx, &ConnectRequest{}); err != nil {
		return d, errors.WithStack(err)
	}

	return d, nil
}

// Info ...
func (t Conn) Info(ctx context.Context) (info *StatusResponse, err error) {
	var (
		_zero StatusRequest
	)
	rpc := NewAgentClient(t.conn)
	if info, err = rpc.Info(ctx, &_zero); err != nil {
		return info, errors.Wrap(err, "failed to retrieve info")
	}

	return info, nil
}

// Watch for messages sent to the leader. blocks.
func (t Conn) Watch(ctx context.Context, out chan<- *Message) (err error) {
	var (
		src Quorum_WatchClient
		msg *Message
	)
	debugx.Println("watch started")
	defer debugx.Println("watch finished")

	c := NewQuorumClient(t.conn)
	if src, err = c.Watch(ctx, &WatchRequest{}); err != nil {
		return errors.WithStack(err)
	}

	for msg, err = src.Recv(); err == nil; msg, err = src.Recv() {
		out <- msg
	}

	return errorsx.Compact(errors.WithStack(err), src.CloseSend())
}

// Dispatch messages to the leader.
func (t Conn) Dispatch(ctx context.Context, messages ...*Message) (err error) {
	var (
		out = DispatchRequest{
			Messages: messages,
		}
	)

	c := NewQuorumClient(t.conn)

	if _, err = c.Dispatch(ctx, &out); err != nil {
		return err
	}

	return nil
}

// Logs return the logs for the given deployment.
func (t Conn) Logs(ctx context.Context, p *Peer, did []byte) io.ReadCloser {
	var (
		err error
		c   Agent_LogsClient
	)

	rpc := NewAgentClient(t.conn)
	if c, err = rpc.Logs(ctx, &LogRequest{Peer: p, DeploymentID: did}); err != nil {
		return io.NopCloser(iox.ErrReader(err))
	}

	return readLogs(c)
}

type logsClient interface {
	Recv() (*LogResponse, error)
}

func readLogs(c logsClient) io.ReadCloser {
	r, w := io.Pipe()
	go func() {
		for {
			var (
				werr error
				resp *LogResponse
			)

			if resp, werr = c.Recv(); werr != nil {
				w.CloseWithError(werr)
				return
			}

			if _, werr = w.Write(resp.Content); werr != nil {
				w.CloseWithError(werr)
				return
			}
		}
	}()

	return r
}

type archiveWriter interface {
	Send(*UploadChunk) error
}

func streamArchive(src io.Reader, stream archiveWriter) (err error) {
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
