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
		log.Println("force refreshing peers due to missing leader")
		t.protocol.SetPeers(t.raftp.getPeers(c))
	}

	return nextState
}

func (t peer) refreshPeers() bool {
	const (
		gracePeriod = 30 * time.Second
	)

	if t.protocol.Leader() != "" {
		return false
	}

	if t.protocol.LastContact().Add(gracePeriod).After(time.Now()) {
		return false
	}

	return true
}
