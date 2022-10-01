package agentutil

import (
	"context"

	"github.com/james-lawrence/bw/agent"
	"github.com/james-lawrence/bw/agent/dialers"
	"github.com/pkg/errors"
	"google.golang.org/grpc"
)

type operation interface {
	Visit(context.Context, *agent.Peer, agent.Client) error
}

// Operation a pure function operation to apply to the entire cluster.
type Operation func(ctx context.Context, p *agent.Peer, c agent.Client) error

// Visit - implements the operation interface.
func (t Operation) Visit(ctx context.Context, p *agent.Peer, c agent.Client) error {
	return t(ctx, p, c)
}

type peers interface {
	Peers() []*agent.Peer
}

// PeerSet a set of static peers.
type PeerSet []*agent.Peer

// Peers the set of peers.
func (t PeerSet) Peers() []*agent.Peer {
	return t
}

// NewClusterOperation applies an operation to every node in the cluster.
func NewClusterOperation(ctx context.Context, o operation) func(peers, dialers.Defaults) error {
	return func(c peers, d dialers.Defaults) (err error) {
		for _, peer := range c.Peers() {
			if err = dialAndVisit(ctx, d, peer, o); err != nil {
				return err
			}
		}

		return nil
	}
}

func dialAndVisit(ctx context.Context, d dialers.Defaults, p *agent.Peer, o operation) (err error) {
	var (
		conn *grpc.ClientConn
	)

	dd := dialers.NewDirect(agent.RPCAddress(p), d.Defaults()...)
	if conn, err = dd.DialContext(ctx); err != nil {
		return errors.WithStack(err)
	}
	defer conn.Close()

	return errors.WithStack(o.Visit(ctx, p, agent.NewConn(conn)))
}
