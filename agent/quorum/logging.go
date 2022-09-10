package quorum

import (
	"io"
	"log"

	"github.com/davecgh/go-spew/spew"
	"github.com/james-lawrence/bw/agent"
)

// Logging transcoder
type Logging struct{}

// Decode logs the message received by the state machine.
func (t Logging) Decode(_ TranscoderContext, m *agent.Message) error {
	log.Println("transcoding", spew.Sdump(m))
	return nil
}

// Encode noop.
func (t Logging) Encode(io.Writer) error {
	return nil
}
