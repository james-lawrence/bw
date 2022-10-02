package termui

import (
	"context"
	"time"

	"github.com/james-lawrence/bw/agent"
	"github.com/james-lawrence/bw/agent/dialers"
	"github.com/james-lawrence/bw/internal/contextx"
	"github.com/james-lawrence/bw/ux"
)

func NewFromClientConfig(ctx context.Context, c agent.ConfigClient, d dialers.Defaults, local *agent.Peer, events chan *agent.Message) {
	dctx, ddone := context.WithTimeout(ctx, c.DeployTimeout+time.Minute)
	New(dctx, ddone, d, local, events)
}

func New(ctx context.Context, shutdown context.CancelFunc, c dialers.Defaults, local *agent.Peer, events chan *agent.Message) {
	contextx.WaitGroupAdd(ctx, 1)

	go func() {
		defer shutdown()
		defer contextx.WaitGroupDone(ctx)
		ux.Deploy(ctx, events, ux.OptionFailureDisplay(ux.NewFailureDisplayPrint(local, c)))
	}()
}

func NewLogging(ctx context.Context, shutdown context.CancelFunc, events chan *agent.Message, options ...ux.Option) {
	contextx.WaitGroupAdd(ctx, 1)

	go func() {
		defer shutdown()
		defer contextx.WaitGroupDone(ctx)
		ux.Logging(ctx, events, options...)
	}()
}
