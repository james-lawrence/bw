package systemx

import (
	"context"
	"os"
	"os/signal"
	"sync"
)

// Cleanup - waits for one of the provided signals, or for the provided context's
// done event to be received. Once received the cleanup function is executed and
// blocks while it waits for everything to finish
func Cleanup(ctx context.Context, cancel func(), wg *sync.WaitGroup, sigs ...os.Signal) func(func()) {
	return func(cleanup func()) {
		signals := make(chan os.Signal, 1)
		signal.Notify(signals, sigs...)
		defer close(signals)
		defer signal.Stop(signals)

		select {
		case <-ctx.Done():
		case _ = <-signals:
			cancel()
		}

		cleanup()
		wg.Wait()
	}
}
