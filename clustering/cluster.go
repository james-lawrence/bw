package clustering

import (
	"context"
	"net"
	"time"

	"github.com/hashicorp/memberlist"
	"github.com/james-lawrence/bw/clustering/rendezvous"
)

// Interfaces used by this package to perform various work.

// cluster is used to extract the peers within a cluster from the underlying implementations.
type Joiner interface {
	Join(...string) (int, error)
	Members() []*memberlist.Node
}

// Rendezvous interface
type Rendezvous interface {
	Members() []*memberlist.Node
	Get(key []byte) *memberlist.Node
	GetN(n int, key []byte) []*memberlist.Node
}

type LocalRendezvous interface {
	Rendezvous
	LocalNode() *memberlist.Node
}

type Replication interface {
	UpdateNode(timeout time.Duration) error
}

// C interface to a cluster.
type C interface {
	Joiner
	Rendezvous
	Replication
	Shutdown() error
	LocalNode() *memberlist.Node
}

// snapshotter is used to persist the peers within a cluster.
type snapshotter interface {
	Snapshot([]string) error
}

// Source is used to pull peers from various sources, typically from a snapshot.
// used when bootstraping a cluster.
type Source interface {
	Peers(context.Context) ([]string, error)
}

// Memberlist represents the cluster.
type Memberlist struct {
	config *memberlist.Config
	list   *memberlist.Memberlist
}

// Leave ...
func (t Memberlist) Leave(d time.Duration) (err error) {
	if t.list == nil {
		return nil
	}

	if cause := t.list.Leave(d); cause != nil {
		err = cause
	}

	if cause := t.list.Shutdown(); err == nil && cause != nil {
		err = cause
	}

	return err
}

// Join ...
func (t Memberlist) Join(existing ...string) (int, error) {
	return t.list.Join(existing)
}

// Config returns the configuration for the cluster.
func (t Memberlist) Config() *memberlist.Config {
	return t.config
}

// Members returns the members of the cluster.
func (t Memberlist) Members() []*memberlist.Node {
	return t.list.Members()
}

// Get - computes the peer that is responsible for the given key.
func (t Memberlist) Get(key []byte) *memberlist.Node {
	return rendezvous.Max(key, t.Members())
}

// GetN - computes the top N peer for the given key.
func (t Memberlist) GetN(n int, key []byte) []*memberlist.Node {
	return rendezvous.MaxN(n, key, t.Members())
}

// IsLocal - checks if the local peer is responsible for the given key,
// returns true, and the local peer if the local peer is responsible.
// returns false, and the peer that is responsible for the key otherwise.
func (t Memberlist) IsLocal(key []byte) (bool, *memberlist.Node) {
	n := t.Get(key)
	return n == t.list.LocalNode(), n
}

// LocalNode returns the local node of the cluster.
func (t Memberlist) LocalNode() *memberlist.Node {
	return t.list.LocalNode()
}

// LocalAddr returns the local node's IP address.
func (t Memberlist) LocalAddr() net.IP {
	return t.LocalNode().Addr
}

// Shutdown - leaves the cluster.
func (t Memberlist) Shutdown() error {
	return t.Leave(3 * time.Second)
}

// ForceShutdown -- do not use.
func (t Memberlist) ForceShutdown() error {
	return t.list.Shutdown()
}

// GetPrimaryKey ...
func (t Memberlist) GetPrimaryKey() []byte {
	return t.config.Keyring.GetPrimaryKey()
}

func (t Memberlist) UpdateNode(timeout time.Duration) error {
	return t.list.UpdateNode(timeout)
}
