package agent

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"io"
	"log"
	"net"
	"strconv"

	"google.golang.org/grpc"

	"bitbucket.org/jatone/bearded-wookie/deployment/agent"
	"github.com/pkg/errors"
)

// DialClient ...
func DialClient(address string, options ...grpc.DialOption) (_ignored Client, err error) {
	var (
		conn *grpc.ClientConn
	)

	if conn, err = grpc.Dial(address, options...); err != nil {
		return _ignored, errors.Wrap(err, "failed to connect to peer")
	}

	return Client{conn: conn}, nil
}

// Client ...
type Client struct {
	conn *grpc.ClientConn
}

// Close ...
func (t Client) Close() error {
	return t.conn.Close()
}

// Upload ...
func (t Client) Upload(srcbytes uint64, src io.Reader) (info agent.Archive, err error) {
	var (
		stream agent.Agent_UploadClient
		_info  *agent.Archive
	)

	rpc := agent.NewAgentClient(t.conn)
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
func (t Client) Deploy(info agent.Archive) error {
	var (
		err error
	)

	rpc := agent.NewAgentClient(t.conn)
	if _, err = rpc.Deploy(context.Background(), &info); err != nil {
		return errors.Wrap(err, "failed to initiated deploy")
	}

	return nil
}

// Credentials ...
func (t Client) Credentials() (agent.Peer, []string, []byte, error) {
	var (
		err      error
		_zeroReq agent.DetailsRequest
		response *agent.Details
	)
	rpc := agent.NewAgentClient(t.conn)
	if response, err = rpc.Quorum(context.Background(), &_zeroReq); err != nil {
		return agent.Peer{}, []string(nil), nil, errors.WithStack(err)
	}

	peers := make([]string, 0, len(response.Quorum))
	quorum := make([]agent.Peer, 0, len(response.Quorum))
	for _, p := range response.Quorum {
		peers = append(peers, net.JoinHostPort(p.Ip, strconv.Itoa(int(p.SWIMPort))))
		quorum = append(quorum, *p)
	}

	return quorum[0], peers, response.Secret, nil
}

// Info ...
func (t Client) Info() (_zeroInfo agent.Status, err error) {
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
func (t Client) Watch(out chan<- agent.Message) (err error) {
	var (
		src agent.Quorum_WatchClient
		msg *agent.Message
	)
	log.Println("watch started")
	defer log.Println("watch finished")

	c := agent.NewQuorumClient(t.conn)
	if src, err = c.Watch(context.Background(), &agent.WatchRequest{}); err != nil {
		log.Println("failed to connect to server", err)
		return errors.WithStack(err)
	}

	log.Println("receiving messages")
	for msg, err = src.Recv(); err == nil; msg, err = src.Recv() {
		log.Println("message received")
		out <- *msg
	}

	return errors.WithStack(err)
}

// Dispatch messages to the leader.
func (t Client) Dispatch(messages ...agent.Message) (err error) {
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

func (t Client) streamArchive(src io.Reader, stream agent.Agent_UploadClient) (err error) {
	buf := make([]byte, 0, 1024*1024)
	emit := func(chunk, checksum []byte) error {
		log.Println("writing chunk", len(chunk))
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
