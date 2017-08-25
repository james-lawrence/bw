package clustering

import (
	"net"
	"time"

	"bitbucket.org/jatone/bearded-wookie/clustering/rendezvous"
	"github.com/hashicorp/memberlist"
)

// Interfaces used by this package to perform various work.

// cluster is used to extract the peers within a cluster from the underlying implementations.
type cluster interface {
	Members() []*memberlist.Node
}

// snapshotter is used to persist the peers within a cluster.
type snapshotter interface {
	Snapshot([]string) error
}

// peering is used to pull peers from various sources, typically from a snapshot.
// used when bootstraping a cluster.
type peering interface {
	Peers() ([]string, error)
}

// Cluster represents the cluster.
type Cluster struct {
	list *memberlist.Memberlist
}

// Members returns the members of the cluster.
func (t Cluster) Members() []*memberlist.Node {
	return t.list.Members()
}

// Get - computes the peer that is responsible for the given key.
func (t Cluster) Get(key []byte) *memberlist.Node {
	return rendezvous.Max(key, t.Members())
}

// GetN - computes the top N peer for the given key.
func (t Cluster) GetN(n int, key []byte) []*memberlist.Node {
	return rendezvous.MaxN(n, key, t.Members())
}

// IsLocal - checks if the local peer is responsible for the given key,
// returns true, and the local peer if the local peer is responsible.
// returns false, and the peer that is responsible for the key otherwise.
func (t Cluster) IsLocal(key []byte) (bool, *memberlist.Node) {
	n := t.Get(key)
	return n == t.list.LocalNode(), n
}

// LocalNode returns the local node of the cluster.
func (t Cluster) LocalNode() *memberlist.Node {
	return t.list.LocalNode()
}

// LocalAddr returns the local node's IP address.
func (t Cluster) LocalAddr() net.IP {
	return t.LocalNode().Addr
}

// Shutdown - leaves the cluster.
func (t Cluster) Shutdown() error {
	t.list.Leave(3 * time.Second)
	return t.list.Shutdown()
}
