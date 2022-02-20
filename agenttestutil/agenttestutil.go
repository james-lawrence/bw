package agenttestutil

import (
	"fmt"
	"net"

	"github.com/james-lawrence/bw/agent"
	"github.com/james-lawrence/bw/cluster"
	"github.com/james-lawrence/bw/clustering"
)

// NewPeers ...
func NewPeers(n int) (peers []agent.Peer) {
	for i := 0; i < n; i++ {
		ip := net.ParseIP(fmt.Sprintf("127.0.0.%d", i+1))
		peers = append(peers, *agent.NewPeer(ip.String(), agent.PeerOptionIP(ip)))
	}

	return peers
}

// NewCluster ...
func NewCluster(p *agent.Peer, opts ...clustering.Option) (c cluster.Cluster, err error) {
	var (
		cp clustering.Memberlist
	)
	local := cluster.NewLocal(p)
	ip := net.ParseIP(p.Ip)
	opts = append(
		opts,
		clustering.OptionNodeID(ip.String()),
		clustering.OptionBindAddress(ip.String()),
		clustering.OptionBindPort(int(local.Peer.P2PPort)),
	)

	if cp, err = clustering.NewCluster(opts...); err != nil {
		return c, err
	}

	return cluster.New(local, cp), nil

}
