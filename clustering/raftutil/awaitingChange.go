package raftutil

import (
	"sync"

	"bitbucket.org/jatone/bearded-wookie/x/debugx"
)

type conditionTransition struct {
	next state
	cond *sync.Cond
}

func (t conditionTransition) Update(c cluster) state {
	debugx.Println("locking condition")
	t.cond.L.Lock()
	debugx.Println("waiting condition")
	t.cond.Wait()
	debugx.Println("unlocking condition")
	t.cond.L.Unlock()
	return t.next
}
