package agentutil

import (
	"context"

	"github.com/james-lawrence/bw/agent"
	"github.com/james-lawrence/bw/agent/dialers"
	"github.com/james-lawrence/bw/internal/grpcx"
)

// Cancel - cancels all deploys across the cluster.
func Cancel(ctx context.Context, c peers, d dialers.Defaulted) error {
	return NewClusterOperation(ctx, Operation(func(ctx context.Context, p *agent.Peer, c agent.Client) error {
		if cause := c.NodeCancel(ctx); grpcx.IgnoreShutdownErrors(cause) != nil {
			return cause
		}

		return nil
	}))(c, d)
}
