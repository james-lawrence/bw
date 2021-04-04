package agentutil

import (
	"context"
	"io"
	"time"

	"github.com/james-lawrence/bw/agent"
	"github.com/james-lawrence/bw/agent/dialers"
	"github.com/james-lawrence/bw/internal/x/iox"
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

		w.CloseWithError(NewClusterOperation(Operation(func(c agent.Client) error {
			// TODO: change cluster operation to provide peer.
			return PrintLogs(ctx, c, nil, deploymentID, w)
		}))(c, d))
	}()
	return r
}
