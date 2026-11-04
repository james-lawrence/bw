package agentutil

import (
	"context"

	"github.com/james-lawrence/bw/agent"
	"github.com/james-lawrence/bw/agent/dialers"
	"github.com/james-lawrence/bw/internal/errorsx"
	"google.golang.org/grpc"
	"google.golang.org/grpc/stats"
)

func NewConnWatcher(ctx context.Context, events chan<- *agent.Message) grpc.DialOption {
	return grpc.WithStatsHandler(dialers.ConnWatcher(func(c context.Context, s stats.ConnStats) {
		remote := agent.NewPeerFromTemplate(
			errorsx.ZeroSilent(agent.NewPeerFromURI(dialers.Address(c))),
		)

		switch s.(type) {
		case *stats.ConnBegin:
			select {
			case <-c.Done():
			case <-ctx.Done():
			case events <- agent.NewConnectionLog(remote, agent.ConnectionEvent_Connected, ""):
			}
		case *stats.ConnEnd:
			select {
			case <-c.Done():
			case <-ctx.Done():
			case events <- agent.NewConnectionLog(remote, agent.ConnectionEvent_Disconnected, ""):
			}
		}
	}))
}
