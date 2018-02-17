package raftutil

import (
	"log"
	"time"

	"github.com/hashicorp/raft"
)

type peer struct {
	raftp    *Protocol
	protocol *raft.Raft
	initTime time.Time
}

func (t peer) lastContact() time.Time {
	var (
		lastContact = t.protocol.LastContact()
	)

	// if lastContact is a zero time. then we've never had a leader.
	// when this is the case fallback to the initTime of the this peer.
	if lastContact.IsZero() {
		return t.initTime
	}

	return lastContact
}

func (t peer) Update(c cluster) state {
	var (
		nextState state = conditionTransition{
			next: t,
			cond: t.raftp.ClusterChange,
		}
	)

	switch s := t.protocol.State(); s {
	case raft.Leader:
		return leader{
			raftp:    t.raftp,
			protocol: t.protocol,
		}.Update(c)
	default:
		log.Println("peer current state", s)
		if maybeLeave(c) || t.raftp.deadlockedLeadership(t.protocol, t.lastContact()) {
			leave(t.raftp, t.protocol)
			return conditionTransition{
				next: passive{
					raftp: t.raftp,
				},
				cond: t.raftp.ClusterChange,
			}
		}
	}

	return nextState
}
