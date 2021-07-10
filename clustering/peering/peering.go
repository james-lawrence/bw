package peering

import (
	"context"
	"net"
)

// Closure - allows for a peering strategy that is contained within a function.
type Closure func(context.Context) ([]string, error)

// Peers - returns the results of the closure.
func (t Closure) Peers(ctx context.Context) ([]string, error) {
	return t(ctx)
}

// NewStaticTCP built a static set of peers from TCPAddr.
func NewStaticTCP(peers ...*net.TCPAddr) Static {
	addresses := make([]string, 0, len(peers))
	for _, addr := range peers {
		addresses = append(addresses, addr.String())
	}

	return NewStatic(addresses...)
}

// NewStatic converts a set of peers into a peering strategy
func NewStatic(peers ...string) Static {
	return Static{peers: peers}
}

// Static ...
type Static struct {
	peers []string
}

// Peers - returns the set of peers.
func (t Static) Peers(context.Context) ([]string, error) {
	return t.peers, nil
}
