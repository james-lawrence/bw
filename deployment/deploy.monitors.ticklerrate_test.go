package deployment

import (
	"context"
	"sync"
	"testing"
	"time"
)

func TestMonitorTicklerPeriodic(t *testing.T) {
	t.Run("signals repeatedly until the context is done", func(t *testing.T) {
		ctx, cancel := context.WithCancel(t.Context())
		defer cancel()

		var (
			tickle = sync.NewCond(&sync.Mutex{})
			done   = make(chan struct{})
		)

		go func() {
			defer close(done)
			MonitorTicklerPeriodic(5*time.Millisecond)(ctx, tickle, nil)
		}()

		// a signal only reaches a goroutine that is already waiting.
		for range 3 {
			signaled := waiter(tickle)
			select {
			case <-signaled:
			case <-time.After(2 * time.Second):
				t.Fatal("expected the tickler to signal")
			}
		}

		cancel()
		select {
		case <-done:
		case <-time.After(2 * time.Second):
			t.Fatal("expected the tickler to stop")
		}
	})
}
