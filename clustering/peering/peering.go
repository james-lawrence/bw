package peering

import "github.com/hashicorp/memberlist"

// cluster is used to extract the peers within a cluster from the underlying implementations.
type cluster interface {
	Members() []*memberlist.Node
}

// Closure - allows for a peering strategy that is contained within a function.
type Closure func() ([]string, error)

// Peers - returns the results of the closure.
func (t Closure) Peers() ([]string, error) {
	return t()
}
