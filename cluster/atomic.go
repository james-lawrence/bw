package cluster

import (
	"github.com/james-lawrence/bw/agent"
	"github.com/james-lawrence/bw/internal/atomicx"
)

// NewAtomic cluster whose quorum can be swapped safely while in use.
func NewAtomic(local *agent.Peer, quorum ...*agent.Peer) Atomic {
	return Atomic{
		local:  local,
		quorum: atomicx.NewValue(&quorum),
	}
}

// Atomic is a cluster with a quorum that can be replaced concurrently.
type Atomic struct {
	local  *agent.Peer
	quorum atomicx.V[*[]*agent.Peer]
}

// Local ...
func (t Atomic) Local() *agent.Peer {
	return t.local
}

// Quorum returns a copy of the current quorum.
func (t Atomic) Quorum() []*agent.Peer {
	return append([]*agent.Peer(nil), *t.quorum.Load()...)
}

// Update replaces the quorum.
func (t Atomic) Update(quorum ...*agent.Peer) {
	t.quorum.Store(&quorum)
}
