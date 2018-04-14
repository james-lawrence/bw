package agentutil

import (
	"github.com/james-lawrence/bw/agent"
)

// Cancel - cancels all deploys across the cluster.
func Cancel(c peers, d agent.Dialer) error {
	return NewClusterOperation(Operation(func(c agent.Client) error {
		if cause := c.Cancel(); cause != nil {
			return cause
		}

		return nil
	}))(c, d)
}
