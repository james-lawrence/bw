package raftutil

import (
	"log"
	"time"

	"github.com/hashicorp/raft"
	"github.com/james-lawrence/bw/internal/debugx"
	"github.com/james-lawrence/bw/internal/stringsx"
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
	log.Println("current leader", stringsx.DefaultIfBlank(leader, "[None]"), t.lastContact, time.Since(t.lastContact), ">", t.protocol.lastContactGrace)
	if leader == "" && t.lastContact.Add(t.protocol.lastContactGrace).Before(time.Now()) {
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
		return leader(t).Update(c)
	default:
		debugx.Println("peer current state", s)
		if t.protocol.MaybeLeave(c) || t.deadleadership(t.r) {
			return leave(t.stateMeta)
		}

		return conditionTransition{
			next: t.updateLastContact(t.r, time.Now()),
			cond: t.protocol.ClusterChange,
		}
	}
}
