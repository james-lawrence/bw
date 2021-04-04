package agentutil

import (
	"github.com/james-lawrence/bw/agent"
	"github.com/james-lawrence/bw/agent/dialers"
)

// Cancel - cancels all deploys across the cluster.
func Cancel(c peers, d dialers.Defaulted) error {
	return NewClusterOperation(Operation(func(c agent.Client) error {
		if cause := c.NodeCancel(); cause != nil {
			return cause
		}

		return nil
	}))(c, d)
}
