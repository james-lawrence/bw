package raftutil

import (
	"log"
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

	log.Printf("CONDITION TRANSITION: %T\n", t.next)

	return t.next
}

type delayedTransition struct {
	next state
	time.Duration
}

func (t delayedTransition) Update(c cluster) state {
	time.Sleep(t.Duration)

	log.Printf("TIMED TRANSITION: %T\n", t.next)

	return t.next
}
