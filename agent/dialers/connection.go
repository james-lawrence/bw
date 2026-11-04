package dialers

import (
	"context"

	"google.golang.org/grpc/stats"
)

type HandleConn func(ctx context.Context, s stats.ConnStats)

func ConnWatcher(c HandleConn) *clogger {
	return &clogger{c: c}
}

type clogger struct {
	c HandleConn
}

func (l *clogger) TagConn(ctx context.Context, info *stats.ConnTagInfo) context.Context {
	if info == nil {
		return ctx
	}

	return WithAddress(ctx, info.RemoteAddr.String())
}

func (l *clogger) HandleConn(ctx context.Context, s stats.ConnStats) {
	l.c(ctx, s)
}

func (l *clogger) TagRPC(ctx context.Context, info *stats.RPCTagInfo) context.Context {
	return ctx
}

func (l *clogger) HandleRPC(ctx context.Context, s stats.RPCStats) {}
