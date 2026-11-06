// Package jsonl writes protobuf messages as json lines.
package jsonl

import (
	"bufio"
	"bytes"
	"io"

	"google.golang.org/protobuf/encoding/protojson"
	"google.golang.org/protobuf/proto"
)

// Encoder writes each message as a single line of json.
type Encoder struct {
	out io.Writer
}

// NewEncoder returns an Encoder writing to w.
func NewEncoder(w io.Writer) *Encoder {
	return &Encoder{out: w}
}

// Encode writes each message followed by a newline.
func (t *Encoder) Encode(msgs ...proto.Message) error {
	for _, m := range msgs {
		b, err := protojson.MarshalOptions{}.Marshal(m)
		if err != nil {
			return err
		}

		if _, err = t.out.Write(append(b, '\n')); err != nil {
			return err
		}
	}

	return nil
}

// Decoder reads the lines written by Encoder.
type Decoder struct {
	in *bufio.Reader
}

// NewDecoder returns a Decoder reading from r.
func NewDecoder(r io.Reader) *Decoder {
	return &Decoder{in: bufio.NewReader(r)}
}

// Decode reads the next non blank line into m, returning io.EOF once there are no more lines.
func (t *Decoder) Decode(m proto.Message) error {
	for {
		line, err := t.in.ReadBytes('\n')
		if len(bytes.TrimSpace(line)) > 0 {
			return protojson.UnmarshalOptions{}.Unmarshal(line, m)
		}

		if err != nil {
			return err
		}
	}
}
