package deploy

import (
	"context"
	"sync/atomic"
	"time"

	"github.com/james-lawrence/bw/agent"
)

// Progress periodically reports the upload progress to the events channel. progress is client local
// and must never be dispatched to the cluster. any event resets the ux's heartbeat timer which
// prevents the connection from being considered dead during long uploads.
// the returned counter should be incremented with the bytes uploaded, see iox.CountingReader.
// the returned function must be called when the upload finishes. it stops the reporting and,
// when the upload succeeded, emits the final completed progress event.
func Progress(ctx context.Context, local *agent.Peer, events chan<- *agent.Message, total int64) (uploaded *atomic.Uint64, completed func(error)) {
	uploaded = new(atomic.Uint64)

	var (
		pctx, cancel = context.WithCancel(ctx)
		done         = make(chan struct{})
	)

	go func() {
		defer close(done)

		t := time.NewTicker(time.Second)
		defer t.Stop()

		for {
			select {
			case <-pctx.Done():
				return
			case <-t.C:
				select {
				case events <- agent.NewUploadProgress(local, uploaded.Load(), uint64(total)):
				default:
				}
			}
		}
	}()

	return uploaded, func(err error) {
		cancel()
		<-done

		if err != nil {
			return
		}

		select {
		case events <- agent.NewUploadProgress(local, uint64(total), uint64(total)):
		case <-ctx.Done():
		}
	}
}
