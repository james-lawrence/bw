package quorum

import (
	"context"
	"io"
	"time"

	"github.com/james-lawrence/bw/agent"
	"github.com/james-lawrence/bw/internal/x/logx"
	"github.com/pkg/errors"
)

// NewObserver creates an observer of the state machine.
func NewObserver(c chan agent.Message) Observer {
	return Observer{
		c: c,
	}
}

// Observer used to observe messages processed by the state machine.
type Observer struct {
	c chan agent.Message
}

// Observe consume the results of the observer by forwarding them to the ConnectableDispatcher
func (t Observer) Observe(ctx context.Context, d agent.ConnectableDispatcher) {
	for m := range t.c {
		ctx, cancel := context.WithTimeout(ctx, 10*time.Second)
		logx.MaybeLog(errors.Wrap(d.Dispatch(ctx, m), "failed to deliver dispatched event to watchers"))
		cancel()
	}
}

// Decode consume the messages passing them to the buffer.
func (t Observer) Decode(ctx TranscoderContext, m agent.Message) error {
	if m.Hidden || ctx.State == StateRecovering {
		return nil
	}

	t.c <- m

	return nil
}

// Encode satisfy the transcoder interface. does nothing.
func (t Observer) Encode(dst io.Writer) (err error) {
	return nil
}
