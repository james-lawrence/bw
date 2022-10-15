package termui

import (
	"context"
	"time"

	"github.com/james-lawrence/bw/agent"
	"github.com/james-lawrence/bw/agent/dialers"
	"github.com/james-lawrence/bw/agentutil"
	"github.com/james-lawrence/bw/internal/contextx"
	"github.com/james-lawrence/bw/ux"
)

func NewFromClientConfig(ctx context.Context, config agent.ConfigClient, d dialers.Quorum, local *agent.Peer, events chan *agent.Message, options ...ux.Option) {
	dctx, ddone := context.WithTimeout(ctx, config.Deployment.Timeout+time.Minute)
	New(dctx, ddone, d, local, events, options...)
}

func New(ctx context.Context, shutdown context.CancelFunc, d dialers.Quorum, local *agent.Peer, events chan *agent.Message, options ...ux.Option) {
	contextx.WaitGroupAdd(ctx, 1)
	cached := dialers.NewCached(d)
	go agentutil.WatchEvents(ctx, local, cached, events)
	go func() {
		defer shutdown()
		defer contextx.WaitGroupDone(ctx)
		ux.Deploy(
			ctx, cached, events,
			append(
				options,
				ux.OptionFailureDisplay(ux.NewFailureDisplayPrint(local, cached)),
			)...,
		)
	}()
}

func NewLogging(ctx context.Context, shutdown context.CancelFunc, d dialers.Quorum, local *agent.Peer, events chan *agent.Message, options ...ux.Option) {
	contextx.WaitGroupAdd(ctx, 1)
	cached := dialers.NewCached(d)

	go agentutil.WatchEvents(ctx, local, cached, events)
	go func() {
		defer shutdown()
		defer contextx.WaitGroupDone(ctx)
		ux.Logging(ctx, cached, events, options...)
	}()
}
