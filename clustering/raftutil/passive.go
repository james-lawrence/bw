package raftutil

import (
	"context"
	"log"
	"sync"
	"time"

	"github.com/james-lawrence/bw/agent"
	"github.com/james-lawrence/bw/internal/x/contextx"

	"github.com/hashicorp/raft"
	"github.com/pkg/errors"
)

type passive struct {
	protocol *Protocol
	sgroup   *sync.WaitGroup
}

func (t passive) Update(c rendezvous) state {
	var (
		err       error
		r         *raft.Raft
		transport raft.Transport
	)

	unstable := delayedTransition{
		next:     t,
		Duration: time.Second,
	}

	quorum := t.protocol.isMember(c)
	maintainState := conditionTransition{
		next: t,
		cond: t.protocol.ClusterChange,
	}

	if !quorum {
		return maintainState
	}

	log.Println(t.protocol.LocalNode.Name, "promoting self into raft protocol")

	if transport, r, err = t.protocol.connect(c); err != nil {
		log.Println(errors.Wrap(err, "failed to join raft protocol remaining in current state"))
		return unstable
	}

	ctx, done := context.WithCancel(t.protocol.Context)
	sm := stateMeta{
		r:           r,
		q:           backlogQueueWorker{Queue: make(chan *agent.ClusterWatchEvents, 100)},
		transport:   transport,
		protocol:    t.protocol,
		sgroup:      t.sgroup,
		lastContact: time.Now(),
		ctx:         ctx,
		done:        done,
	}

	if r.LastIndex() == 0 {
		if err = r.BootstrapCluster(configuration(c)).Error(); err != nil {
			log.Println("raft bootstrap failed", r.LastIndex(), err)
			sm.cleanShutdown()
			return unstable
		}
	}

	if err = sm.connect(); err != nil {
		sm.cleanShutdown()
		return unstable
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
