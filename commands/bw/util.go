package main

import (
	"github.com/james-lawrence/bw"
	"github.com/james-lawrence/bw/agent"
	"github.com/james-lawrence/bw/x/systemx"
)

// newClientPeer create a client peer.
func newClientPeer(options ...agent.PeerOption) (p agent.Peer) {
	return agent.NewPeerFromTemplate(
		agent.Peer{
			Name:   bw.MustGenerateID().String(),
			Ip:     systemx.HostnameOrLocalhost(),
			Status: agent.Peer_Client,
		},
		agent.PeerOptionStatus(agent.Peer_Client),
	)
}
