package cluster

import (
	"log"
	"sync"

	"github.com/hashicorp/memberlist"
)

// NewEvents ...
func NewEvents(l *sync.Cond) Events {
	return Events{
		b: l,
	}
}

// Events ...
type Events struct {
	b *sync.Cond
}

// NotifyJoin is invoked when a node is detected to have joined.
// The Node argument must not be modified.
func (t Events) NotifyJoin(peer *memberlist.Node) {
	log.Println("Join:", peer.Name, peer.Addr.String(), int(peer.Port))
	t.b.Broadcast()
}

// NotifyLeave is invoked when a node is detected to have left.
// The Node argument must not be modified.
func (t Events) NotifyLeave(peer *memberlist.Node) {
	log.Println("Leave:", peer.Name, peer.Addr.String(), int(peer.Port))
	t.b.Broadcast()
}

// NotifyUpdate is invoked when a node is detected to have
// updated, usually involving the meta data. The Node argument
// must not be modified.
func (t Events) NotifyUpdate(peer *memberlist.Node) {
	log.Println("Update:", peer.Name, peer.Addr.String(), int(peer.Port))
}
