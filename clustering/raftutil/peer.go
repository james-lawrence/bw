package raftutil

import (
	"log"

	"github.com/hashicorp/raft"
)

type peer struct {
	raftp    Protocol
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

	log.Println("current state", t.protocol.State())
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

	return nextState
}
