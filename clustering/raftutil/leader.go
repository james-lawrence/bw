package raftutil

import (
	"log"
	"time"

	"github.com/james-lawrence/bw/x/debugx"

	"github.com/hashicorp/memberlist"
	"github.com/hashicorp/raft"
)

type leader struct {
	raftp    *Protocol
	protocol *raft.Raft
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
		if t.cleanupPeers(c.LocalNode(), possiblePeers(c)...) {
			go t.raftp.unstable(time.Second)
		}
		return maintainState
	default:
		log.Println("lost leadership: demoting to peer")
		return peer{
			raftp:    t.raftp,
			protocol: t.protocol,
			initTime: time.Now(),
		}.Update(c)
	}
}

// cleanupPeers returns true if the peer set was unstable.
func (t leader) cleanupPeers(local *memberlist.Node, candidates ...*memberlist.Node) (unstable bool) {
	config := t.protocol.GetConfiguration()
	if err := config.Error(); err != nil {
		log.Println("failed to retrieve peers", err)
		return true
	}

	// remove self from peer set.
	peers := removePeer(raft.ServerID(local.Name), config.Configuration().Servers...)
	log.Println("candidates", peersToString(t.raftp.Port, candidates...))
	log.Println("peers", peers)

	for _, peer := range candidates {
		id := raft.ServerID(peer.Name)
		p := raft.ServerAddress(peerToString(t.raftp.Port, peer))
		peers = removePeer(id, peers...)
		if err := t.protocol.AddVoter(id, p, t.protocol.GetConfiguration().Index(), time.Second).Error(); err != nil {
			log.Println("failed to add peer", err)
			unstable = true
		}
	}

	for _, peer := range peers {
		if err := t.protocol.RemoveServer(peer.ID, t.protocol.GetConfiguration().Index(), time.Second).Error(); err != nil {
			log.Println("failed to remove peer", err)
			unstable = true
		}
	}

	return unstable
}

func removePeer(id raft.ServerID, peers ...raft.Server) []raft.Server {
	result := make([]raft.Server, 0, len(peers))
	for _, peer := range peers {
		if peer.ID == id {
			continue
		}
		result = append(result, peer)
	}

	return result
}
