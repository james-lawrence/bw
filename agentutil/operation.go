package agentutil

import (
	"github.com/james-lawrence/bw/agent"
	"github.com/pkg/errors"
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
func NewClusterOperation(o operation) func(peers, agent.Dialer) error {
	return func(c peers, dialer agent.Dialer) (err error) {
		for _, peer := range c.Peers() {
			if err = dialAndVisit(dialer, peer, o); err != nil {
				return err
			}
		}

		return nil
	}
}

func dialAndVisit(dialer agent.Dialer, p agent.Peer, o operation) (err error) {
	var (
		cx agent.Client
	)

	if cx, err = dialer.Dial(p); err != nil {
		return errors.WithStack(err)
	}
	defer cx.Close()

	return errors.WithStack(o.Visit(cx))
}
