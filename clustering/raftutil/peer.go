package raftutil

import (
	"log"
	"time"

	"bitbucket.org/jatone/bearded-wookie/x/debugx"

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
	debugx.Println("peer update invoked")
	debugx.Println("current leader", t.protocol.Leader(), t.protocol.LastContact().Format(time.Stamp))

	switch s := t.protocol.State(); s {
	case raft.Leader:
		nextState = leader{
			raftp:    t.raftp,
			protocol: t.protocol,
			peers:    t.peers,
		}.Update(c)
	case raft.Follower:
		if t.refreshPeers() {
			log.Println("force refreshing peers due to missing leader")
			t.protocol.SetPeers(t.raftp.getPeers(c))
		}
	default:
		debugx.Println("current state", s)
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

func (t peer) refreshPeers() bool {
	const (
		gracePeriod = 30 * time.Second
	)
	deadline := t.protocol.LastContact().Add(gracePeriod)

	debugx.Println("deadline", deadline.Format(time.Stamp), "now", time.Now().Format(time.Stamp), deadline.Before(time.Now()))
	if deadline.Before(time.Now()) {
		return true
	}

	return false
}
