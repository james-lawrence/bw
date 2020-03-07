package raftutil

import (
	"log"
	"time"

	"github.com/hashicorp/raft"
)

type peer struct {
	stateMeta
}

func (t peer) lastContact() time.Time {
	var (
		lastContact = t.r.LastContact()
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
		maintain state = conditionTransition{
			next:     t,
			cond:     t.protocol.ClusterChange,
			Duration: t.protocol.PassiveCheckin,
		}
	)

	switch s := t.r.State(); s {
	case raft.Leader:
		return leader{
			stateMeta: t.stateMeta,
		}.Update(c)
	default:
		log.Println(c.LocalNode().Name, "peer current state", s)
		if maybeLeave(c) || t.protocol.deadlockedLeadership(c.LocalNode(), t.r, t.lastContact()) {
			return leave(t, t.stateMeta)
		}
		log.Println(c.LocalNode().Name, "peer state updated", s)
	}

	return maintain
}
