package agentutil

import (
	"github.com/james-lawrence/bw/agent"
	"github.com/james-lawrence/bw/agent/dialers"
)

// Shutdown runs the shutdown command on the entire cluster.
func Shutdown(c peers, d dialers.Defaults) error {
	return NewClusterOperation(Operation(func(c agent.Client) error {
		if cause := c.Shutdown(); cause != nil {
			return cause
		}

		return nil
	}))(c, d)
}
