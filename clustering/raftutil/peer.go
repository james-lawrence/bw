package raftutil

import (
	"log"
	"time"

	"github.com/hashicorp/raft"
)

type peer struct {
	raftp    *Protocol
	protocol *raft.Raft
	peers    raft.PeerStore
}

func (t peer) Update(c cluster) state {
	var (
		nextState state = conditionTransition{
			next: t,
			cond: t.raftp.ClusterChange,
		}
	)

	switch t.protocol.State() {
	case raft.Leader:
		nextState = leader{
			raftp:    t.raftp,
			protocol: t.protocol,
			peers:    t.peers,
		}.Update(c)
	}

	if maybeLeave(t.protocol, c) {
		nextState = conditionTransition{
			next: passive{
				raftp: t.raftp,
			},
			cond: t.raftp.ClusterChange,
		}
	}

	if t.refreshPeers() {
		peers := t.raftp.getPeers(c)
		log.Println("force refreshing peers due to missing leader. self:", peersToString(t.raftp.Port, c.LocalNode()), "peers:", peers)
		t.protocol.SetPeers(peers)
	}

	return nextState
}

func (t peer) refreshPeers() bool {
	const (
		gracePeriod = 5 * time.Second
	)

	if err := t.protocol.VerifyLeader().Error(); err != nil {
		log.Println("verify leader", err)
	}

	if t.protocol.Leader() != "" {
		log.Println("leader is empty")
		return false
	}

	if t.protocol.LastContact().Add(gracePeriod).After(time.Now()) {
		return false
	}

	return true
}
