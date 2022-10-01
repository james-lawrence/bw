package agentutil

import (
	"context"

	"github.com/james-lawrence/bw/agent"
	"github.com/james-lawrence/bw/agent/dialers"
)

// Shutdown runs the shutdown command on the entire cluster.
func Shutdown(c peers, d dialers.Defaults) error {
	return NewClusterOperation(context.Background(), Operation(func(ctx context.Context, p *agent.Peer, c agent.Client) error {
		if cause := c.Shutdown(ctx); cause != nil {
			return cause
		}

		return nil
	}))(c, d)
}
