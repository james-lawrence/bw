// Package discovery is used to provide information
// about the system to anyone.
package discovery

import (
	"context"

	"github.com/james-lawrence/bw/agent"
)

// cluster interface for the package.
type cluster interface {
	Quorum() []agent.Peer
}

type authorization interface {
	Authorized(context.Context) error
}

func peerToNode(p agent.Peer) Node {
	return Node{
		Ip:            p.Ip,
		Name:          p.Name,
		RPCPort:       p.RPCPort,
		RaftPort:      p.RaftPort,
		SWIMPort:      p.SWIMPort,
		TorrentPort:   p.TorrentPort,
		DiscoveryPort: p.DiscoveryPort,
	}
}

// nodeToPeer ...
func nodeToPeer(n Node) agent.Peer {
	return agent.Peer{
		Ip:            n.Ip,
		Name:          n.Name,
		RPCPort:       n.RPCPort,
		RaftPort:      n.RaftPort,
		SWIMPort:      n.SWIMPort,
		TorrentPort:   n.TorrentPort,
		DiscoveryPort: n.DiscoveryPort,
	}
}
