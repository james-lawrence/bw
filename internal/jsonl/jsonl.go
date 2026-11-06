// Package jsonl writes protobuf messages as json lines.
package jsonl

import (
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
