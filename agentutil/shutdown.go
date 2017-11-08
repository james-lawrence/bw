package agentutil

import (
	"bitbucket.org/jatone/bearded-wookie/agent"
	"google.golang.org/grpc"
)

// Shutdown runs the shutdown command on the entire cluster.
func Shutdown(c peers, options ...grpc.DialOption) error {
	return NewClusterOperation(Operation(func(c agent.Client) error {
		if cause := c.Shutdown(); cause != nil {
			return cause
		}

		return nil
	}))(c, options...)
}
