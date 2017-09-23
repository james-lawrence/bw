package raftutil

import (
	"log"

	"bitbucket.org/jatone/bearded-wookie/x/debugx"

	"github.com/hashicorp/memberlist"
	"github.com/hashicorp/raft"
)

type leader struct {
	raftp    *Protocol
	protocol *raft.Raft
	peers    raft.PeerStore
}

func (t leader) Update(c cluster) state {
	maintainState := conditionTransition{
		next: t,
		cond: t.raftp.ClusterChange,
	}

	switch t.protocol.State() {
	case raft.Leader:
		t.cleanupPeers(possiblePeers(c)...)
		if maybeLeave(t.protocol, c) {
			return conditionTransition{
				next: passive{
					raftp: t.raftp,
				},
				cond: t.raftp.ClusterChange,
			}
		}
		return maintainState
	case raft.Follower:
		return peer{
			raftp:    t.raftp,
			protocol: t.protocol,
			peers:    t.peers,
		}.Update(c)
	default:
		return passive{
			raftp: t.raftp,
		}.Update(c)
	}
}

func (t leader) cleanupPeers(peers ...*memberlist.Node) {
	leaders, err := t.peers.Peers()
	if err != nil {
		log.Println("failed to retrieve peers", err)
		return
	}

	debugx.Println("peers", peersToString(t.raftp.Port, peers...))
	debugx.Println("leaders", leaders)
	for _, peer := range peers {
		p := peerToString(t.raftp.Port, peer)
		if raft.PeerContained(leaders, p) {
			leaders = raft.ExcludePeer(leaders, p)
			continue
		}

		if err = t.protocol.AddPeer(p).Error(); err != nil {
			log.Println("failed to add peer", err)
		}
	}

	debugx.Println("dead nodes", leaders)
	for _, peer := range leaders {
		if err = t.protocol.RemovePeer(peer).Error(); err != nil {
			log.Println("failed to remove peer", err)
		}
	}
}
