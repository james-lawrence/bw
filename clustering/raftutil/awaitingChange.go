package raftutil

import (
	"context"
	"sync"
	"time"
)

type conditionTransition struct {
	timeout time.Duration
	next    state
	cond    *sync.Cond
}

func (t conditionTransition) Update(c rendezvous) state {
	ctx, done := context.WithCancel(context.Background())
	defer done()

	if t.timeout > 0 {
		go func() {
			select {
			case <-time.After(t.timeout):
				t.cond.Broadcast()
			case <-ctx.Done():
			}
		}()
	}

	t.cond.L.Lock()
	t.cond.Wait()
	t.cond.L.Unlock()

	return t.next
}

func delayed(next state, cond *sync.Cond, t time.Duration) conditionTransition {
	return conditionTransition{
		timeout: t,
		next:    next,
		cond:    cond,
	}
}
