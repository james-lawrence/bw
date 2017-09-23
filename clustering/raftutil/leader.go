package raftutil

import (
	"log"
	"time"

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
	var (
		maintainState state = conditionTransition{
			next: t,
			cond: t.raftp.ClusterChange,
		}
	)

	debugx.Println("leader update invoked")
	switch t.protocol.State() {
	case raft.Leader:
		debugx.Println("still the leader")
		if t.cleanupPeers(possiblePeers(c)...) {
			go t.raftp.unstable(time.Second)
		}

		if maybeLeave(t.protocol, c) {
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

// cleanupPeers returns true if the peer set was unstable.
func (t leader) cleanupPeers(candidates ...*memberlist.Node) (unstable bool) {
	peers, err := t.peers.Peers()
	if err != nil {
		log.Println("failed to retrieve peers", err)
		return true
	}

	debugx.Println("candidates", peersToString(t.raftp.Port, candidates...))
	debugx.Println("peers", peers)

	for _, peer := range candidates {
		p := peerToString(t.raftp.Port, peer)
		if raft.PeerContained(peers, p) {
			peers = raft.ExcludePeer(peers, p)
			continue
		}

		if err = t.protocol.AddPeer(p).Error(); err != nil {
			log.Println("failed to add peer", err)
		}

		unstable = true
	}

	debugx.Println("dead nodes", peers)
	for _, peer := range peers {
		if err = t.protocol.RemovePeer(peer).Error(); err != nil {
			log.Println("failed to remove peer", err)
		}

		unstable = true
	}

	return unstable
}
