package quorum_test

import (
	"io"
	"log"

	"github.com/james-lawrence/bw/agent"
	"github.com/james-lawrence/bw/agent/quorum"
	"os"
	"testing"
)

func TestMain(m *testing.M) {
	log.SetOutput(io.Discard)
	os.Exit(m.Run())
}

// NewEvery creates an observer of the state machine.
func NewEvery(c chan *agent.Message) Every {
	return Every{
		c: c,
	}
}

// Every used to observe messages processed by the state machine.
// unlike Observer the intention here is to literally see every message.
type Every struct {
	c chan *agent.Message
}

// Decode consume the messages passing them to the buffer.
func (t Every) Decode(ctx quorum.TranscoderContext, m *agent.Message) error {
	t.c <- m
	return nil
}

// Encode satisfy the transcoder interface. does nothing.
func (t Every) Encode(dst io.Writer) (err error) {
	return nil
}
