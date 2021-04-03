package raftutil

import (
	"context"
	"log"
	"sync"
	"time"

	"github.com/james-lawrence/bw/internal/x/contextx"

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

	ctx, done := context.WithCancel(t.protocol.Context)
	sm := stateMeta{
		r:           r,
		transport:   network,
		protocol:    t.protocol,
		sgroup:      t.sgroup,
		lastContact: time.Now(),
		ctx:         ctx,
		done:        done,
	}

	if r.LastIndex() == 0 {
		if err = r.BootstrapCluster(configuration(c)).Error(); err != nil {
			log.Println("raft bootstrap failed", r.LastIndex(), err)
			sm.cleanShutdown(c)
			return maintainState
		}
	}

	// add this to the parent context waitgroup
	contextx.WaitGroupAdd(t.protocol.Context, 1)
	go sm.waitShutdown(c)

	sm.sgroup.Add(1)
	go sm.background()

	return peer{
		stateMeta: sm,
	}.Update(c)
}
