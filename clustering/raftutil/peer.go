package raftutil

import (
	"log"
	"time"

	"github.com/james-lawrence/bw/x/debugx"

	"github.com/hashicorp/raft"
)

type peer struct {
	raftp    *Protocol
	protocol *raft.Raft
}

func (t peer) Update(c cluster) state {
	var (
		nextState state = conditionTransition{
			next: t,
			cond: t.raftp.ClusterChange,
		}
	)

	debugx.Println("peer update invoked")
	log.Println("current leader", t.protocol.Leader(), t.protocol.LastContact().Format(time.Stamp))

	switch s := t.protocol.State(); s {
	case raft.Leader:
		return leader{
			raftp:    t.raftp,
			protocol: t.protocol,
		}.Update(c)
	default:
		log.Println("peer current state", s)
		if maybeLeave(t.protocol, c) {
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
