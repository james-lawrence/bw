package quorum

import (
	"context"
	"io"

	"github.com/james-lawrence/bw/agent"
	"github.com/james-lawrence/bw/internal/errorsx"
)

// NewObserver creates an observer of the state machine.
func NewObserver(d agent.Dispatcher) Observer {
	return Observer{
		d: d,
	}
}

// Observer used to observe messages processed by the state machine.
type Observer struct {
	d agent.Dispatcher
}

// Decode record the message in the journal; observers pull from it at their own pace.
// hidden messages are recorded, they are filtered by the readers. messages replayed while the
// WAL is recovering are not, they have already been journaled.
func (t Observer) Decode(ctx TranscoderContext, m *agent.Message) error {
	if ctx.State == StateRecovering {
		return nil
	}

	return errorsx.Compact(t.d.Dispatch(context.Background(), m))
}

// Encode satisfy the transcoder interface. does nothing.
func (t Observer) Encode(dst io.Writer) (err error) {
	return nil
}
