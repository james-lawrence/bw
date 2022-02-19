package peering

import (
	"context"

	"github.com/hashicorp/memberlist"
	"github.com/james-lawrence/bw/agent/dialers"
	"github.com/james-lawrence/bw/agent/discovery"
)

type P2P struct {
	Address string
	Dialer  dialers.Defaults
}

func (t P2P) Peers(ctx context.Context) (results []string, err error) {
	var (
		nodes []*memberlist.Node
	)

	if nodes, err = discovery.Snapshot(t.Address, t.Dialer.Defaults()...); err != nil {
		return nil, err
	}

	for _, n := range nodes {
		results = append(results, n.Address())
	}

	return results, nil
}
