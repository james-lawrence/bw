package agentutil

import (
	"bitbucket.org/jatone/bearded-wookie/deployment/agent"
	"github.com/pkg/errors"
	"google.golang.org/grpc"
)

type cluster interface {
	Peers() []agent.Peer
}

type operation interface {
	Visit(client) error
}

type operationFunc func(c client) error

func (t operationFunc) Visit(c client) error {
	return t(c)
}

// NewClusterOperation applies an operation to every node in the cluster.
func NewClusterOperation(o operation) func(cluster, ...grpc.DialOption) error {
	return func(c cluster, options ...grpc.DialOption) (err error) {
		var (
			cx client
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
