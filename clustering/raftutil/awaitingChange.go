package raftutil

import (
	"sync"
	"time"

	"github.com/james-lawrence/bw/internal/x/debugx"
)

type conditionTransition struct {
	next state
	cond *sync.Cond
	time.Duration
}

func (t conditionTransition) Update(c cluster) state {
	xx := time.NewTimer(t.Duration)
	defer xx.Stop()
	go func() {
		<-xx.C
		t.cond.Broadcast()
	}()

	t.cond.L.Lock()
	t.cond.Wait()
	t.cond.L.Unlock()

	debugx.Printf("CONDITION TRANSITION: %T %v\n", t.next, t.Duration)
	return t.next
}

type delayedTransition struct {
	next state
	time.Duration
}

func (t delayedTransition) Update(c cluster) state {
	time.Sleep(t.Duration)
	debugx.Printf("TIMED TRANSITION: %T\n", t.next)
	return t.next
}
