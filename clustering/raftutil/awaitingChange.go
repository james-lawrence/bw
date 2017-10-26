package raftutil

import (
	"sync"
	"time"
)

type conditionTransition struct {
	next state
	cond *sync.Cond
}

func (t conditionTransition) Update(c cluster) state {
	t.cond.L.Lock()
	t.cond.Wait()
	t.cond.L.Unlock()
	return t.next
}

type delayedTransition struct {
	next state
	time.Duration
}

func (t delayedTransition) Update(c cluster) state {
	time.Sleep(t.Duration)
	return t.next
}
