// Package asyncx provides small concurrency primitives.
package asyncx

// Semaphore is a counting semaphore with a fixed capacity. unlike a sync.Cond it remembers
// broadcasts that happen while nobody is waiting (up to its capacity), and unlike a sync.Cond
// waiting on it can be combined with other channels in a select, timers, contexts, etc.
//
// the zero value is not usable, use NewSemaphore.
type Semaphore struct {
	tokens chan struct{}
}

// NewSemaphore returns a semaphore that holds at most n tokens, starting with none.
// a capacity of 1 gives a binary semaphore, useful as a sticky wakeup signal: any number
// of broadcasts before the next wait collapse into a single wakeup.
func NewSemaphore(n int) *Semaphore {
	if n < 1 {
		panic("asyncx: semaphore capacity must be at least 1")
	}

	return &Semaphore{tokens: make(chan struct{}, n)}
}

// Signal adds a token without blocking. reports false when the semaphore is already full,
// in which case the broadcast is dropped.
func (t *Semaphore) Signal() bool {
	select {
	case t.tokens <- struct{}{}:
		return true
	default:
		return false
	}
}

// Wait returns a channel that yields a token, for use in a select. receiving from the
// channel acquires the token.
func (t *Semaphore) Wait() <-chan struct{} {
	return t.tokens
}
