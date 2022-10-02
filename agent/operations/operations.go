package operations

import (
	"context"

	"github.com/hashicorp/memberlist"
	"github.com/james-lawrence/bw/agent"
	"github.com/james-lawrence/bw/agent/dialers"
	"github.com/pkg/errors"
	"google.golang.org/grpc"
)

type peers interface {
	Members() []*memberlist.Node
}

type operation interface {
	Visit(context.Context, *agent.Peer, grpc.ClientConnInterface) error
}

type Fn func(context.Context, *agent.Peer, grpc.ClientConnInterface) error

func (t Fn) Visit(ctx context.Context, p *agent.Peer, conn grpc.ClientConnInterface) error {
	return t(ctx, p, conn)
}

// New applies an operation to every node in the cluster.
func New(ctx context.Context, o operation) func(peers, dialers.Defaults) error {
	return func(c peers, d dialers.Defaults) (err error) {
		for _, peer := range agent.NodesToPeers(c.Members()...) {
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

	return errors.WithStack(o.Visit(ctx, p, conn))
}
