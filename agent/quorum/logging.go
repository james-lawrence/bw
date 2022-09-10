package quorum

import (
	"io"
	"log"

	"github.com/davecgh/go-spew/spew"
	"github.com/james-lawrence/bw"
	"github.com/james-lawrence/bw/agent"
	"github.com/james-lawrence/bw/internal/x/envx"
)

// Logging transcoder
type Logging struct{}

// Decode logs the message received by the state machine.
func (t Logging) Decode(_ TranscoderContext, m *agent.Message) error {
	if envx.Boolean(false, bw.EnvLogsQuorum, bw.EnvLogsVerbose) {
		log.Println("transcoding", spew.Sdump(m))
	}

	return nil
}

// Encode noop.
func (t Logging) Encode(io.Writer) error {
	return nil
}
