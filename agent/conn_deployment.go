package agent

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"io"
	"time"

	"github.com/pkg/errors"
	"golang.org/x/time/rate"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	status "google.golang.org/grpc/status"

	"github.com/james-lawrence/bw/internal/errorsx"
	"github.com/james-lawrence/bw/internal/grpcx"
	"github.com/james-lawrence/bw/internal/iox"
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
func (t DeployConn) Cancel(ctx context.Context, req *CancelRequest) error {
	_, err := NewDeploymentsClient(t.conn).Cancel(ctx, req)
	return errors.WithStack(err)
}

// Upload an archive to be deployed.
func (t DeployConn) Upload(ctx context.Context, m *UploadMetadata, src io.Reader) (info *Archive, err error) {
	var (
		stream Deployments_UploadClient
		_info  *UploadResponse
		rpc    = NewDeploymentsClient(t.conn)
	)

	if stream, err = rpc.Upload(ctx); err != nil {
		return info, errors.Wrap(err, "failed to create upload stream")
	}

	initialChunk := &UploadChunk{
		Checksum: []byte{},
		Data:     []byte{},
		InitialChunkMetadata: &UploadChunk_Metadata{
			Metadata: m,
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
func (t DeployConn) RemoteDeploy(ctx context.Context, initiator string, dopts *DeployOptions, a *Archive, peers ...*Peer) (err error) {
	l := rate.NewLimiter(rate.Every(time.Second), 1)
	rpc := NewDeploymentsClient(t.conn)
	req := DeployCommandRequest{
		Initiator: initiator,
		Archive:   a,
		Options:   dopts,
		Peers:     peers,
	}

	for err = status.Error(codes.Unavailable, ""); grpcx.IsUnavailable(err); err = errorsx.Compact(l.Wait(ctx), err) {
		_, err = rpc.Deploy(ctx, &req)
	}

	return errors.WithStack(err)
}

// Watch for messages sent to the leader. blocks.
func (t DeployConn) Watch(ctx context.Context, out chan<- *Message) (err error) {
	var (
		src Deployments_WatchClient
		msg *Message
	)

	c := NewDeploymentsClient(t.conn)
	if src, err = c.Watch(ctx, &WatchRequest{}); err != nil {
		return errors.WithStack(err)
	}

	for msg, err = src.Recv(); err == nil; msg, err = src.Recv() {
		select {
		case out <- msg:
		case <-ctx.Done():
			return errors.WithStack(ctx.Err())
		}
	}

	return errorsx.Compact(errors.WithStack(err), src.CloseSend())
}

// Logs return the logs for the given deployment.
func (t DeployConn) Logs(ctx context.Context, p *Peer, did []byte) io.ReadCloser {
	var (
		err error
		c   Deployments_LogsClient
	)

	rpc := NewDeploymentsClient(t.conn)
	if c, err = rpc.Logs(ctx, &LogRequest{Peer: p, DeploymentID: did}); err != nil {
		return io.NopCloser(iox.ErrReader(err))
	}

	return readLogs(c)
}
