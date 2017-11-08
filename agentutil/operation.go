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

type peers interface {
	Peers() []agent.Peer
}

// PeerSet a set of static peers.
type PeerSet []agent.Peer

// Peers the set of peers.
func (t PeerSet) Peers() []agent.Peer {
	return t
}

// NewClusterOperation applies an operation to every node in the cluster.
func NewClusterOperation(o operation) func(peers, ...grpc.DialOption) error {
	return func(c peers, options ...grpc.DialOption) (err error) {
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
