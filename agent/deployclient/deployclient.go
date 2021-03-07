// Package deployclient provides functionality for the deploy client.
package deployclient

import (
	"log"

	"github.com/hashicorp/memberlist"
	"github.com/james-lawrence/bw/agent"
	"github.com/james-lawrence/bw/agentutil"
)

// NewClusterEventHandler ...
func NewClusterEventHandler(bus chan *agent.Message) ClusterEventHandler {
	return ClusterEventHandler{
		bus: bus,
	}
}

// ClusterEventHandler ...
type ClusterEventHandler struct {
	bus chan *agent.Message
}

// NotifyJoin logs when a peer joins.
func (t ClusterEventHandler) NotifyJoin(peer *memberlist.Node) {
	t.update(peer)
}

// NotifyLeave logs when a peer leaves.
func (t ClusterEventHandler) NotifyLeave(peer *memberlist.Node) {
	t.update(peer, agent.PeerOptionStatus(agent.Peer_Gone))
}

// NotifyUpdate logs when a peer updates.
func (t ClusterEventHandler) NotifyUpdate(peer *memberlist.Node) {
	t.update(peer)
}

func (t ClusterEventHandler) update(peer *memberlist.Node, options ...agent.PeerOption) {
	var (
		err error
		p   *agent.Peer
	)

	if p, err = agent.NodeToPeer(peer); err != nil {
		log.Println("failed to convert memberlist.Node to peer", err)
		return
	}

	t.bus <- agentutil.PeerEvent(agent.NewPeerFromTemplate(p, options...))
}
