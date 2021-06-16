package quorum

import (
	"context"
	"io"
	"log"
	"time"

	"github.com/james-lawrence/bw/agent"
	"github.com/james-lawrence/bw/internal/x/errorsx"
)

// NewObserver creates an observer of the state machine.
func NewObserver(d agent.ConnectableDispatcher) Observer {
	return Observer{
		d: d,
	}
}

// Observer used to observe messages processed by the state machine.
type Observer struct {
	d agent.ConnectableDispatcher
}

// Decode consume the messages passing them to observers
func (t Observer) Decode(ctx TranscoderContext, m *agent.Message) error {
	if m.Hidden {
		return nil
	}

	if ctx.State == StateRecovering {
		log.Println("unable to observe WAL messages, recovering")
		return nil
	}

	dctx, done := context.WithTimeout(context.Background(), 3*time.Second)
	err := t.d.Dispatch(dctx, m)
	done()
	return errorsx.Compact(err)
}

// Encode satisfy the transcoder interface. does nothing.
func (t Observer) Encode(dst io.Writer) (err error) {
	return nil
}
