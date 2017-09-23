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
	debugx.Println("leader update invoked")
	switch t.protocol.State() {
	case raft.Leader:
		debugx.Println("still the leader")
		t.cleanupPeers(possiblePeers(c)...)
		debugx.Println("cleaned peers")
		if maybeLeave(t.protocol, c) {
			log.Println("leader is leaving raft cluster")
			return conditionTransition{
				next: passive{
					raftp: t.raftp,
				},
				cond: t.raftp.ClusterChange,
			}
		}
		return maintainState
	default:
		log.Println("lost leadership: demoting to peer")
		return peer{
			raftp:    t.raftp,
			protocol: t.protocol,
			peers:    t.peers,
		}.Update(c)
	}
}

func (t leader) cleanupPeers(peers ...*memberlist.Node) {
	candidates, err := t.peers.Peers()
	if err != nil {
		log.Println("failed to retrieve peers", err)
		return
	}

	debugx.Println("peers", peersToString(t.raftp.Port, peers...))
	debugx.Println("candidates", candidates)
	for _, peer := range peers {
		p := peerToString(t.raftp.Port, peer)
		if !raft.PeerContained(candidates, p) {
			log.Println()
			if err = t.protocol.AddPeer(p).Error(); err != nil {
				log.Println("failed to add peer", err)
			}
		} else {
			candidates = raft.ExcludePeer(candidates, p)
		}
	}

	debugx.Println("dead nodes", candidates)
	for _, peer := range candidates {
		if err = t.protocol.RemovePeer(peer).Error(); err != nil {
			log.Println("failed to remove peer", err)
		}
	}
}
