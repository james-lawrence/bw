package raftutil

import (
	"sync"
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
