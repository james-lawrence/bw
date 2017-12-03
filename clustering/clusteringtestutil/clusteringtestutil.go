package clusteringtestutil

import (
	"fmt"
	"net"

	"github.com/james-lawrence/bw/clustering"

	"github.com/hashicorp/memberlist"
)

// NewCluster ...
func NewCluster(n int, opts ...clustering.Option) (out []clustering.Cluster, err error) {
	for i := 0; i < n; i++ {
		var (
			c clustering.Cluster
		)
		ip := net.ParseIP(fmt.Sprintf("127.0.0.%d", i+1))
		opts = append(
			opts,
			clustering.OptionNodeID(ip.String()),
			clustering.OptionBindAddress(ip.String()),
		)

		if c, err = clustering.NewCluster(opts...); err != nil {
			ShutdownCluster(out...)
			return out, err
		}

		out = append(out, c)
	}

	return out, nil
}

// ShutdownCluster ...
func ShutdownCluster(nodes ...clustering.Cluster) (err error) {
	for _, n := range nodes {
		if cause := n.Shutdown(); cause != nil && err == nil {
			err = cause
		}
	}

	return err
}

// NewPeers generates up to 254 peers with IPs
// between 127.0.0.1 and 127.0.0.n
func NewPeers(n int) []*memberlist.Node {
	if n >= 255 {
		panic("only supports generating a cluster up to 254 nodes")
	}

	peers := make([]*memberlist.Node, 0, n)
	for i := 0; i < n; i++ {
		peers = append(peers, NewPeer(fmt.Sprintf("node-%d", i+1), net.ParseIP(fmt.Sprintf("127.0.0.%d", i+1))))
	}

	return peers
}

// NewPeer creates a peer with the given name, and ip.
func NewPeer(name string, ip net.IP) *memberlist.Node {
	return &memberlist.Node{
		Name: name,
		Addr: ip,
	}
}

// NewMock generates a new mock cluster with n peers.
func NewMock(n int) clustering.Mock {
	peers := NewPeers(n)
	return clustering.NewMock(peers[0], peers[1:]...)
}
