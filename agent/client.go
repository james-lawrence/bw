package agent

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"io"
	"log"

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
func (t Client) Upload(src io.Reader) (info agent.Archive, err error) {
	var (
		stream agent.Agent_UploadClient
		_info  *agent.Archive
	)

	rpc := agent.NewAgentClient(t.conn)

	if stream, err = rpc.Upload(context.Background()); err != nil {
		return info, errors.Wrap(err, "failed to create upload stream")
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

// Info ...
func (t Client) Info() (_zeroInfo agent.AgentInfo, err error) {
	var (
		_zero agent.AgentInfoRequest
		info  *agent.AgentInfo
	)
	rpc := agent.NewAgentClient(t.conn)
	if info, err = rpc.Info(context.Background(), &_zero); err != nil {
		return _zeroInfo, errors.Wrap(err, "failed to initiated deploy")
	}

	return *info, nil
}

func (t Client) streamArchive(src io.Reader, stream agent.Agent_UploadClient) (err error) {
	buf := make([]byte, 0, 1024*1024)
	emit := func(chunk, checksum []byte) error {
		log.Println("writing chunk", len(chunk))
		return errors.Wrap(stream.Send(&agent.ArchiveChunk{
			Checksum: checksum,
			Data:     chunk,
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
