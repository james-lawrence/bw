package raftutil

import (
	"sync"
	"time"

	"github.com/james-lawrence/bw/internal/x/debugx"
)

type conditionTransition struct {
	next state
	cond *sync.Cond
}

func (t conditionTransition) Update(c cluster) state {
	t.cond.L.Lock()
	t.cond.Wait()
	t.cond.L.Unlock()
	debugx.Printf("CONDITION TRANSITION: %T\n", t.next)
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
