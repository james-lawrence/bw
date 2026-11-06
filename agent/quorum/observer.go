package quorum

import (
	"context"
	"io"
	"time"

	"github.com/james-lawrence/bw/agent"
	"github.com/james-lawrence/bw/internal/errorsx"
	"github.com/james-lawrence/bw/internal/timex"
)

// journaled the journal controls the observer uses when the dispatcher provides them.
type journaled interface {
	// Retain keeps the messages written since the time.
	Retain(time.Time)
	// Reset discards the messages, the state they recorded is being replaced.
	Reset() error
}

// NewObserver creates an observer of the state machine.
func NewObserver(d agent.Dispatcher) Observer {
	return Observer{d: d}
}

// Observer used to observe messages processed by the state machine.
type Observer struct {
	d agent.Dispatcher
}

// Decode record the message in the journal; observers pull from it at their own pace.
// hidden messages are recorded, they are filtered by the readers. a snapshot restore replaces the state, so the
// journal is reset and the messages the snapshot replays rebuild it. a journal that understands retention keeps the
// messages since the archive of the last successful deploy was created.
func (t Observer) Decode(ctx TranscoderContext, m *agent.Message) error {
	switch m.GetType() {
	case agent.Message_RestoreEventInitiated:
		if j, ok := t.d.(journaled); ok {
			return j.Reset()
		}
		return nil
	case agent.Message_RestoreEventCompleted:
		return nil
	}

	if err := errorsx.Compact(t.d.Dispatch(context.Background(), m)); err != nil {
		return err
	}

	t.retain(m)

	return nil
}

// retain moves the journal to keep the messages since the archive of the deploy that completed was created, or since
// the deploy completed when the archive has no timestamp.
func (t Observer) retain(m *agent.Message) {
	j, ok := t.d.(journaled)
	if !ok || m.GetDeployCommand().GetCommand() != agent.DeployCommand_Done {
		return
	}

	j.Retain(timex.FirstNonZero(timex.UnixOrZero(m.GetDeployCommand().GetArchive().GetTs()), agent.MessageTime(m)))
}

// Encode satisfy the transcoder interface. does nothing.
func (t Observer) Encode(dst io.Writer) (err error) {
	return nil
}
