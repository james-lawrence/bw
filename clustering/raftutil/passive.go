package raftutil

import (
	"log"

	"github.com/hashicorp/raft"
	"github.com/pkg/errors"
)

type passive struct {
	raftp Protocol
}

func (t passive) Update(c cluster) state {
	var (
		err      error
		store    raft.PeerStore
		protocol *raft.Raft
	)

	// if we're not a leader or something goes wrong during this update process
	// maintain our current state.
	maintainState := conditionTransition{
		next: passive{
			raftp: t.raftp,
		},
		cond: t.raftp.ClusterChange,
	}

	if !isMember(c) {
		return maintainState
	}

	log.Println("promoting self into raft protocol")

	if protocol, store, err = t.raftp.connect(c); err != nil {
		log.Println(errors.Wrap(err, "failed to join raft protocol remaining in current state"))
		return maintainState
	}

	log.Println("initial state", protocol.State())

	return peer{
		raftp:    t.raftp,
		peers:    store,
		protocol: protocol,
	}.Update(c)
}
