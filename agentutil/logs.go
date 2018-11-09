package agentutil

import (
	"io"

	"github.com/james-lawrence/bw/agent"
	"github.com/james-lawrence/bw/x/iox"
)

// PrintLogs for the given deployment ID.
func PrintLogs(did []byte, dst io.Writer) Operation {
	return func(c agent.Client) error {
		return iox.Error(io.Copy(dst, c.Logs(did)))
	}
}

// DeploymentLogs retrieves the logs for the given deployment ID from each server in the cluster.
func DeploymentLogs(c cluster, d dialer, deploymentID []byte) io.ReadCloser {
	r, w := io.Pipe()
	go func() {
		w.CloseWithError(NewClusterOperation(PrintLogs(deploymentID, w))(c, d))
	}()
	return r
}
