package quorum

import (
	"context"

	"github.com/james-lawrence/bw/agent"
	"github.com/james-lawrence/bw/x/grpcx"
	"github.com/pkg/errors"
)

type pbObserver struct {
	dst  agent.Quorum_WatchServer
	done context.CancelFunc
}

func (t pbObserver) send(ctx context.Context, m agent.Message) error {
	done := make(chan error, 1)
	select {
	case <-ctx.Done():
		return ctx.Err()
	case done <- t.dst.Send(&m):
		return <-done
	}
}

func (t pbObserver) Receive(ctx context.Context, messages ...agent.Message) (err error) {
	var (
		cause error
	)

	for _, m := range messages {
		if err = t.send(ctx, m); err != nil {
			if cause = errors.Cause(err); cause == context.Canceled {
				return nil
			}

			t.done()

			if grpcx.IgnoreShutdownErrors(cause) == nil {
				return nil
			}

			return errors.Wrapf(err, "error type %T", cause)
		}
	}

	return nil
}
