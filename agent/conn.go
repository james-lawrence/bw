package agent

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"io"

	"google.golang.org/grpc"

	"github.com/james-lawrence/bw/x/debugx"
	"github.com/james-lawrence/bw/x/errorsx"
	"github.com/pkg/errors"
)

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

// Cancel causes the current deploy (if any) to be cancelled.
func (t Conn) Cancel() (err error) {
	rpc := NewAgentClient(t.conn)
	if _, err = rpc.Cancel(context.Background(), &CancelRequest{}); err != nil {
		return errors.WithStack(err)
	}

	return nil
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

// QuorumInfo returns high level details about the state of the cluster.
func (t Conn) QuorumInfo() (z InfoResponse, err error) {
	var (
		resp *InfoResponse
	)

	rpc := NewQuorumClient(t.conn)
	if resp, err = rpc.Info(context.Background(), &InfoRequest{}); err != nil {
		return z, errors.WithStack(err)
	}

	return *resp, nil
}

// Upload ...
func (t Conn) Upload(initiator string, total uint64, src io.Reader) (info Archive, err error) {
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

// Watch for messages sent to the leader. blocks.
func (t Conn) Watch(ctx context.Context, out chan<- Message) (err error) {
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
		out <- *msg
	}

	return errorsx.Compact(errors.WithStack(err), src.CloseSend())
}

// Dispatch messages to the leader.
func (t Conn) Dispatch(ctx context.Context, messages ...Message) (err error) {
	var (
		out = DispatchRequest{
			Messages: MessagesToPtr(messages...),
		}
	)

	c := NewQuorumClient(t.conn)

	if _, err = c.Dispatch(ctx, &out); err != nil {
		return errors.WithStack(err)
	}

	return nil
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
