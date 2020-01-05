package agent

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"io"

	"github.com/pkg/errors"
	"google.golang.org/grpc"

	"github.com/james-lawrence/bw/internal/x/errorsx"
)

// MaybeDeployConn ...
func MaybeDeployConn(c *grpc.ClientConn, err error) (DeployConn, error) {
	if err != nil {
		return DeployConn{}, err
	}
	return DeployConn{conn: c}, nil
}

// NewDeployConn create a new connection.
func NewDeployConn(c *grpc.ClientConn) DeployConn {
	return DeployConn{conn: c}
}

// DeployConn a connect to deal with deployments.
type DeployConn struct {
	conn *grpc.ClientConn
}

// Close ...
func (t DeployConn) Close() error {
	if t.conn == nil {
		return nil
	}

	return t.conn.Close()
}

// Cancel proxy the cancellation through the quorum nodes.
// this cleans up the raft state in addition to the individual nodes.
func (t DeployConn) Cancel() error {
	return errors.New("not implemented")
	// _, err := NewDeploymentsClient(t.conn).Cancel(context.Background(), &CancelRequest{})
	// return errors.WithStack(err)
}

// Upload ...
func (t DeployConn) Upload(initiator string, total uint64, src io.Reader) (info Archive, err error) {
	var (
		stream Deployments_UploadClient
		_info  *UploadResponse
	)

	rpc := NewDeploymentsClient(t.conn)
	if stream, err = rpc.Upload(context.Background()); err != nil {
		return info, errors.Wrap(err, "failed to create upload stream")
	}

	initialChunk := &UploadChunk{
		Checksum: []byte{},
		Data:     []byte{},
		InitialChunkMetadata: &UploadChunk_Metadata{
			Metadata: &UploadMetadata{
				Bytes:     total,
				Initiator: initiator,
			},
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
func (t DeployConn) RemoteDeploy(dopts DeployOptions, a Archive, peers ...Peer) (err error) {
	rpc := NewDeploymentsClient(t.conn)
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

// Watch for messages sent to the leader. blocks.
func (t DeployConn) Watch(ctx context.Context, out chan<- Message) (err error) {
	var (
		src Deployments_WatchClient
		msg *Message
	)

	c := NewDeploymentsClient(t.conn)
	if src, err = c.Watch(ctx, &WatchRequest{}); err != nil {
		return errors.WithStack(err)
	}

	for msg, err = src.Recv(); err == nil; msg, err = src.Recv() {
		out <- *msg
	}

	return errorsx.Compact(errors.WithStack(err), src.CloseSend())
}

func (t DeployConn) streamArchive(src io.Reader, stream Deployments_UploadClient) (err error) {
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

// Logs return the logs for the given deployment.
func (t DeployConn) Logs(ctx context.Context, did []byte) io.ReadCloser {
	var (
		err error
		c   Agent_LogsClient
	)

	r, w := io.Pipe()
	rpc := NewAgentClient(t.conn)
	if c, err = rpc.Logs(ctx, &LogRequest{DeploymentID: did}); err != nil {
		w.CloseWithError(err)
		return r
	}

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
