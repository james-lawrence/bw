package raftutil

import (
	"log"
	"time"

	"github.com/hashicorp/raft"
)

type peer struct {
	stateMeta
}

func (t peer) deadleadership(p *raft.Raft) bool {
	leader := string(p.Leader())

	// one would expect to be able to use raft's LastContact() function here instead of maintaining our own.
	// however that function doesn't actually return accurate values. it updates every time a protocol message
	// is sent, including candidate promotions. thereby not accurately representing the last contact time with
	// the leadership.
	// lastContact is the time of the previous evaluation that saw a leader, it only advances when the state
	// does (cluster changes), so with a leader present the elapsed time is the evaluation cadence not an outage.
	since := time.Since(t.lastContact)
	if leader != "" {
		log.Println("current leader", leader, "previous evaluation with a leader", since)
		return false
	}

	log.Println("current leader [None] missing for", since, "grace period", t.protocol.lastContactGrace)
	if since > t.protocol.lastContactGrace {
		log.Println("leader is missing and grace period has passed, resetting this peer", t.protocol.lastContactGrace)
		return true
	}

	return false
}

func (t peer) updateLastContact(p *raft.Raft, n time.Time) peer {
	leader := string(p.Leader())
	if leader != "" {
		t.lastContact = n
	}

	return peer{
		stateMeta: t.stateMeta,
	}
}

func (t peer) Update(c rendezvous) state {
	switch s := t.r.State(); s {
	case raft.Leader:
		return leader{stateMeta: t.stateMeta}.Update(c)
	default:
		log.Println("peer current state", s)
		if t.protocol.MaybeLeave(c) || t.deadleadership(t.r) {
			return leave(t.stateMeta)
		}

		return conditionTransition{
			next: t.updateLastContact(t.r, time.Now()),
			cond: t.protocol.ClusterChange,
		}
	}
}
