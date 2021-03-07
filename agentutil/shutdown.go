package agentutil

import (
	"github.com/james-lawrence/bw/agent"
)

// Shutdown runs the shutdown command on the entire cluster.
func Shutdown(c peers, d dialer2) error {
	return NewClusterOperation(Operation(func(c agent.Client) error {
		if cause := c.Shutdown(); cause != nil {
			return cause
		}

		return nil
	}))(c, d)
}
