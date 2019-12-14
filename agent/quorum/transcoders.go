package quorum

import (
	"encoding/binary"
	"io"

	"github.com/golang/protobuf/proto"
	"github.com/james-lawrence/bw/agent"
)

// Different states the WAL can be in.
const (
	StateHealthy int64 = iota
	StateRecovering
)

// TranscoderContext provides details to the transcovers about
// the write ahead log, this allows transcoders like observers to properly react.
type TranscoderContext struct {
	State int64
}

// encoders are used to persist the content.
type encoder interface {
	Encode(dst io.Writer) error
}

// decodes are used to process the message.
type decoder interface {
	Decode(TranscoderContext, agent.Message) error
}

type transcoder interface {
	encoder
	decoder
}

// NewTranscoder create a transcoder from many.
func NewTranscoder(transcoders ...transcoder) Transcoder {
	return Transcoder(transcoders)
}

// Transcoder wraps a set of individual transcoders into a single transcoder.
type Transcoder []transcoder

// Encode the state of the various decoders to the destination.
func (t Transcoder) Encode(dst io.Writer) (err error) {
	for _, c := range t {
		if err := c.Encode(dst); err != nil {
			return err
		}
	}

	return nil
}

// Decode pass the message through each decoder. the first error encountered is
// returned.
func (t Transcoder) Decode(ctx TranscoderContext, m agent.Message) error {
	for _, c := range t {
		if err := c.Decode(ctx, m); err != nil {
			return err
		}
	}

	return nil
}

// Discard transcoder
type Discard struct{ Cause error }

// Decode discards the message
func (t Discard) Decode(_ TranscoderContext, m agent.Message) error {
	return t.Cause
}

// Encode noop.
func (t Discard) Encode(io.Writer) error {
	return t.Cause
}

// Encode fundamental encoding method. creates a byte array representing the proto.Message.
func Encode(m proto.Message) (encoded []byte, err error) {
	if encoded, err = proto.Marshal(m); err != nil {
		return encoded, err
	}

	return encodeRaw(encoded), nil
}

func encodeProtoTo(dst io.Writer, m proto.Message) (err error) {
	var (
		encoded []byte
	)

	if encoded, err = Encode(m); err != nil {
		return err
	}

	if _, err = dst.Write(encoded); err != nil {
		return err
	}

	return nil
}

func encodeTo(dst io.Writer, messages ...agent.Message) (err error) {
	for _, m := range messages {
		if err = encodeProtoTo(dst, &m); err != nil {
			return err
		}
	}

	return nil
}

func encodeRaw(encoded []byte) []byte {
	var (
		buf = make([]byte, 8)
	)

	binary.LittleEndian.PutUint64(buf, uint64(len(encoded)))

	return append(buf, encoded...)
}

// Decode a single proto message from the buffer.
func Decode(src io.Reader, m proto.Message) (err error) {
	var (
		buf []byte
	)

	if buf, err = decodeRaw(src); err != nil {
		return err
	}

	return proto.Unmarshal(buf, m)
}

// DecodeEvery reads all messages from the reader into an array
func DecodeEvery(src io.Reader) (buf []agent.Message, err error) {
	for {
		var decoded agent.Message
		if err = Decode(src, &decoded); err != nil {
			break
		}

		buf = append(buf, decoded)
	}

	if err != io.EOF {
		return buf, err
	}

	return buf, nil
}

func decodeRaw(src io.Reader) (buf []byte, err error) {
	buf = make([]byte, 8)

	if _, err = io.ReadFull(src, buf); err != nil {
		return buf, err
	}

	length := binary.LittleEndian.Uint64(buf)
	buf = make([]byte, length)

	if _, err = io.ReadFull(src, buf); err != nil {
		return buf, err
	}

	return buf, err
}
