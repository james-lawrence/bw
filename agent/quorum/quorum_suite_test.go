package quorum_test

import (
	"io"
	"io/ioutil"
	"log"

	"github.com/james-lawrence/bw/agent"
	"github.com/james-lawrence/bw/agent/quorum"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	"testing"
)

func TestQuorum(t *testing.T) {
	log.SetOutput(ioutil.Discard)
	RegisterFailHandler(Fail)
	RunSpecs(t, "Quorum Suite")
}

// NewEvery creates an observer of the state machine.
func NewEvery(c chan agent.Message) Every {
	return Every{
		c: c,
	}
}

// Every used to observe messages processed by the state machine.
// unlike Observer the intention here is to literally see every message.
type Every struct {
	c chan agent.Message
}

// Decode consume the messages passing them to the buffer.
func (t Every) Decode(ctx quorum.TranscoderContext, m agent.Message) error {
	t.c <- m
	return nil
}

// Encode satisfy the transcoder interface. does nothing.
func (t Every) Encode(dst io.Writer) (err error) {
	return nil
}
