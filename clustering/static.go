package clustering

import (
	"time"

	"github.com/james-lawrence/bw/clustering/rendezvous"

	"github.com/hashicorp/memberlist"
)

// NewStatic a static cluster made up of a set of nodes.
func NewStatic(peers ...*memberlist.Node) Static {
	return Static{
		peers: peers,
	}
}

// Static set of nodes
type Static struct {
	peers []*memberlist.Node
}

// Members - see Cluster.
func (t Static) Members() []*memberlist.Node {
	return t.peers
}

// Get - see Cluster.
func (t Static) Get(key []byte) *memberlist.Node {
	return rendezvous.Max(key, t.Members())
}

// GetN - see Cluster.
func (t Static) GetN(n int, key []byte) []*memberlist.Node {
	return rendezvous.MaxN(n, key, t.Members())
}

// Shutdown noop
func (t Static) Shutdown() error {
	return nil
}

// Join - see cluster (deprecated)
func (t Static) Join(...string) (int, error) {
	return len(t.peers), nil
}

// Leave - see cluster (deprecated)
func (t Static) Leave(time.Duration) (err error) {
	return nil
}

// LocalNode - see cluster (deprecated)
func (t Static) LocalNode() *memberlist.Node {
	return &memberlist.Node{}
}

// Config - see Cluster.
func (t Static) Config() *memberlist.Config {
	return memberlist.DefaultLocalConfig()
}
