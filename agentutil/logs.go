package agentutil

import (
	"context"
	"io"
	"time"

	"github.com/james-lawrence/bw/agent"
	"github.com/james-lawrence/bw/agent/dialers"
	"github.com/james-lawrence/bw/internal/iox"
)

// PrintLogs for the given deployment ID.
func PrintLogs(ctx context.Context, c agent.DeployClient, p *agent.Peer, did []byte, dst io.Writer) error {
	return iox.Error(io.Copy(dst, c.Logs(ctx, p, did)))
}

// DeploymentLogs retrieves the logs for the given deployment ID from each server in the cluster.
func DeploymentLogs(c cluster, d dialers.Defaults, deploymentID []byte) io.ReadCloser {
	r, w := io.Pipe()
	go func() {
		ctx, done := context.WithTimeout(context.Background(), 20*time.Second)
		defer done()

		w.CloseWithError(NewClusterOperation(ctx, Operation(func(ctx context.Context, p *agent.Peer, c agent.Client) error {
			return PrintLogs(ctx, c, p, deploymentID, w)
		}))(c, d))
	}()
	return r
}
