package raftutil

import (
	"sync"
	"time"
)

type conditionTransition struct {
	next state
	cond *sync.Cond
}

func (t conditionTransition) Update(c rendezvous) state {
	// xx := time.NewTimer(t.Duration)
	// done := make(chan struct{})
	// defer close(done)
	// defer xx.Stop()
	// go func() {
	// 	select {
	// 	case <-done:
	// 		return
	// 	case <-xx.C:
	// 		t.cond.Broadcast()
	// 	}
	// }()

	t.cond.L.Lock()
	t.cond.Wait()
	t.cond.L.Unlock()

	return t.next
}

type delayedTransition struct {
	next state
	time.Duration
}

func (t delayedTransition) Update(c rendezvous) state {
	time.Sleep(t.Duration)
	return t.next
}
