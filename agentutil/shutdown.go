package agentutil

import (
	"context"
	"log"

	"github.com/james-lawrence/bw/agent"
	"github.com/james-lawrence/bw/agent/dialers"
	"github.com/james-lawrence/bw/internal/grpcx"
	"google.golang.org/grpc/codes"
)

// Shutdown runs the shutdown command on the entire cluster.
func Shutdown(ctx context.Context, c peers, d dialers.Defaults) error {
	return NewClusterOperation(ctx, Operation(func(ctx context.Context, p *agent.Peer, c agent.Client) error {
		log.Println("shutting down initiated", p.Ip)
		defer log.Println("shutting down completed", p.Ip)

		// retry when unavailable because we're literally shooting nodes in the head.
		// as a result the node we're proxied through is going to be nuked at some point.
		return grpcx.Retry(func() error {
			if cause := c.Shutdown(ctx); cause != nil {
				return cause
			}

			return nil
		}, codes.Unavailable)
	}))(c, d)
}
