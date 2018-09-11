package raftutil

import (
	"log"
	"sync"
	"time"

	"github.com/james-lawrence/bw/x/contextx"

	"github.com/hashicorp/raft"
	"github.com/pkg/errors"
)

type passive struct {
	protocol *Protocol
	sgroup   *sync.WaitGroup
}

func (t passive) Update(c cluster) state {
	var (
		err     error
		r       *raft.Raft
		network raft.Transport
	)

	// log.Println(c.LocalNode().Name, "passive update invoked")
	maintainState := conditionTransition{
		next: t,
		cond: t.protocol.ClusterChange,
	}

	if !isMember(c) {
		return delayedTransition{
			next:     t,
			Duration: t.protocol.PassiveCheckin,
		}
	}

	log.Println(c.LocalNode().Name, "promoting self into raft protocol")

	if network, r, err = t.protocol.connect(c); err != nil {
		log.Println(errors.Wrap(err, "failed to join raft protocol remaining in current state"))
		return maintainState
	}

	if r.LastIndex() == 0 {
		if err = r.BootstrapCluster(configuration(t.protocol, c)).Error(); err != nil {
			log.Println("raft bootstrap failed", r.LastIndex(), err)
			t.protocol.maybeShutdown(c, r, network)
			return maintainState
		}
	}

	sm := stateMeta{
		r:         r,
		transport: network,
		protocol:  t.protocol,
		sgroup:    t.sgroup,
		initTime:  time.Now(),
	}

	// add this to the parent context waitgroup
	contextx.WaitGroupAdd(t.protocol.Context, 1)
	go t.protocol.waitShutdown(c, sm)
	sm.sgroup.Add(1)
	go t.protocol.background(sm)

	return peer{
		stateMeta: sm,
	}.Update(c)
}
