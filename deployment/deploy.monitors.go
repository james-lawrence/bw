package deployment

import (
	"context"
	"log"
	"sync"
	"sync/atomic"
	"time"

	"github.com/james-lawrence/bw"
	"github.com/james-lawrence/bw/agent"
	"github.com/james-lawrence/bw/agent/dialers"
	"github.com/james-lawrence/bw/agentutil"
	"github.com/james-lawrence/bw/internal/errorsx"
	"github.com/pkg/errors"
	"golang.org/x/time/rate"
)

// monitors monitor a peers deployment to determine success or failures.
type monitor interface {
	Await(ctx context.Context, q chan *pending, d dispatcher, c cluster, check operation) error
}

func NewMonitorEvent(l *agent.Peer, d dialers.ContextDialer) EventMonitor {
	return EventMonitor{
		l: l,
		d: d,
	}
}

type EventMonitor struct {
	l *agent.Peer
	d dialers.ContextDialer
}

func (t EventMonitor) keepalive(ctx context.Context, tickle *sync.Cond) {
	keepalineduration := 30 * time.Second
	keepalive := time.NewTicker(keepalineduration)
	defer tickle.Signal()
	defer keepalive.Stop()
	for {
		select {
		case <-keepalive.C:
			tickle.Signal()
		case <-ctx.Done():
			return
		}
	}
}

func (t EventMonitor) tickler(ctx context.Context, tickle *sync.Cond, q chan struct{}) {
	for {
		select {
		case <-ctx.Done():
			return
		default:
		}

		tickle.L.Lock()
		tickle.Wait()
		select {
		case q <- struct{}{}:
		case <-ctx.Done():
			return
		}
		tickle.L.Unlock()
	}
}

func (t EventMonitor) watch(ctx context.Context, tickle *sync.Cond) {
	events := make(chan *agent.Message, 100)
	go agentutil.WatchClusterEvents(ctx, t.d, t.l, events)
	for {
		select {
		case m := <-events:
			switch m.Event.(type) {
			case *agent.Message_Deploy:
				tickle.Signal()
			default:
			}
		case <-ctx.Done():
			return
		}
	}
}

func (t EventMonitor) Await(ctx context.Context, q chan *pending, d dispatcher, c cluster, check operation) error {
	// log.Println("event monitoring initiated")
	// defer log.Println("event monitoring completed")
	performcheck := make(chan struct{})
	outstanding := make([]*pending, 0, 128)
	tickle := sync.NewCond(&sync.Mutex{})
	defer tickle.Signal()
	go t.keepalive(ctx, tickle)
	go t.tickler(ctx, tickle, performcheck)
	go t.watch(ctx, tickle)

	for {
		select {
		case task, ok := <-q:
			if !ok {
				q = nil
				tickle.Signal()
				continue
			}
			outstanding = append(outstanding, task)
			tickle.Signal()
		case <-performcheck:
			// log.Println("checkpoint 4", len(outstanding), "/", cap(outstanding))

			remaining := make([]*pending, 0, 128)
			for _, n := range outstanding {
				if healthcheck(ctx, n, check) == nil {
					continue
				}
				remaining = append(remaining, n)
			}

			outstanding = remaining

			if q == nil && len(outstanding) == 0 {
				return nil
			}
		case <-ctx.Done():
			return ctx.Err()
		}
	}
}

type polling struct{}

func (t polling) Await(ctx context.Context, q chan *pending, d dispatcher, c cluster, check operation) (failed error) {
	// log.Println("polling initiated")
	// defer log.Println("polling completed")
	internal := make(chan *pending, 32)
	blocked := int64(0)

	deadline := time.Now().Add(bw.DefaultDeployTimeout)
	if d, ok := ctx.Deadline(); ok {
		deadline = d
	}
	timeout := time.Until(deadline)

	// pop from the two queues until the deadline expires.
	pop := func(ctx context.Context) (*pending, error) {
		for {
			// log.Println("polling pop", atomic.LoadInt64(&blocked), len(internal), cap(internal), q == nil)
			if atomic.LoadInt64(&blocked) == 0 && len(internal) == 0 && q == nil {
				return nil, context.Canceled
			}

			select {
			case task := <-internal:
				return task, nil
			case task, ok := <-q:
				if !ok {
					q = nil
					continue
				}
				return task, nil
			case <-ctx.Done():
				return nil, ctx.Err()
			}
		}
	}

	limit := rate.NewLimiter(rate.Every(time.Second), 1)
	for failed = limit.Wait(ctx); failed == nil; failed = limit.Wait(ctx) {
		var (
			err  error
			task *pending
		)

		if task, err = pop(ctx); err != nil {
			return errorsx.Timedout(errorsx.Ignore(err, context.Canceled), timeout)
		}

		if err = healthcheck(ctx, task, check); err == nil {
			continue
		}

		log.Printf("failed to check: %s - %T, %s\n", task.Peer.Name, errors.Cause(err), err)

		select {
		case internal <- task:
		default:
			atomic.AddInt64(&blocked, 1)
			go func(task *pending) {
				defer atomic.AddInt64(&blocked, -1)
				internal <- task
			}(task)
		}
	}

	return failed
}
