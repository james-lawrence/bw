package agentutil

import (
	"bitbucket.org/jatone/bearded-wookie/agent"
	"github.com/pkg/errors"
	"google.golang.org/grpc"
)

type operation interface {
	Visit(agent.Client) error
}

// Operation a pure function operation to apply to the entire cluster.
type Operation func(c agent.Client) error

// Visit - implements the operation interface.
func (t Operation) Visit(c agent.Client) error {
	return t(c)
}

// NewClusterOperation applies an operation to every node in the cluster.
func NewClusterOperation(o operation) func(cluster, ...grpc.DialOption) error {
	return func(c cluster, options ...grpc.DialOption) (err error) {
		var (
			cx agent.Client
		)

		for _, peer := range c.Peers() {
			if cx, err = DialPeer(peer, options...); err != nil {
				return errors.WithStack(err)
			}
			defer cx.Close()

			if err = o.Visit(cx); err != nil {
				return errors.WithStack(err)
			}
		}

		return nil
	}
}
