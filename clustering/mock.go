package clustering

import (
	"net"
	"time"

	"github.com/james-lawrence/bw/clustering/rendezvous"

	"github.com/hashicorp/memberlist"
)

// NewMock a fake cluster made up of a set of peers and a local node.
func NewMock(local *memberlist.Node, peers ...*memberlist.Node) Mock {
	return Mock{
		local:  local,
		peers:  append(peers, local),
		config: memberlist.DefaultLocalConfig(),
	}
}

// NewSingleNode creates a cluster that is made up of just a single node.
func NewSingleNode(name string, addr net.IP) Mock {
	local := &memberlist.Node{
		Name: name,
		Addr: addr,
	}
	return NewMock(local)
}

// Mock a fake cluster made up of a set of peers and a local node.
type Mock struct {
	local  *memberlist.Node
	peers  []*memberlist.Node
	config *memberlist.Config
}

// Leave ...
func (t Mock) Leave(time.Duration) error {
	return nil
}

// Join ...
func (t Mock) Join(...string) (int, error) {
	return len(t.peers), nil
}

// Config - see Cluster.
func (t Mock) Config() *memberlist.Config {
	return t.config
}

// Members - see Cluster.
func (t Mock) Members() []*memberlist.Node {
	return t.peers
}

// Get - see Cluster.
func (t Mock) Get(key []byte) *memberlist.Node {
	return rendezvous.Max(key, t.Members())
}

// GetN - see Cluster.
func (t Mock) GetN(n int, key []byte) []*memberlist.Node {
	return rendezvous.MaxN(n, key, t.Members())
}

// IsLocal - see Cluster.
func (t Mock) IsLocal(key []byte) (bool, *memberlist.Node) {
	n := t.Get(key)
	return n == t.local, n
}

// LocalNode - see Cluster.
func (t Mock) LocalNode() *memberlist.Node {
	return t.local
}

// LocalAddr - see Cluster.
func (t Mock) LocalAddr() net.IP {
	return t.local.Addr
}

// Shutdown - see Cluster.
func (t Mock) Shutdown() error {
	return nil
}
