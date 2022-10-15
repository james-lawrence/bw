package deployment

import (
	"context"
	"log"
	"runtime"
	"sync"
	"time"

	"github.com/james-lawrence/bw"
	"github.com/james-lawrence/bw/agent"
	"github.com/james-lawrence/bw/agent/dialers"
	"github.com/james-lawrence/bw/agentutil"
	"github.com/james-lawrence/bw/internal/envx"
	"golang.org/x/time/rate"
)

// monitors monitor a peers deployment to determine success or failures.
type monitor interface {
	Await(ctx context.Context, q chan *pending, d dispatcher, c cluster, check operation, additional ...MonitorTickler) error
}

func NewMonitor(ticklers ...MonitorTickler) Monitor {
	return Monitor{
		ticklers: ticklers,
	}
}

type MonitorTickler func(ctx context.Context, tickle *sync.Cond)

func MonitorTicklerPeriodic(d time.Duration) MonitorTickler {
	return MonitorTicklerRate(rate.NewLimiter(rate.Every(d), 1))
}

func MonitorTicklerPeriodicAuto(d time.Duration) MonitorTickler {
	return MonitorTicklerPeriodic(d / 4)
}

func MonitorTicklerRate(r *rate.Limiter) MonitorTickler {
	return func(ctx context.Context, tickle *sync.Cond) {
		defer tickle.Signal()
		for err := r.Wait(ctx); err == nil; err = r.Wait(ctx) {
			if envx.Boolean(false, bw.EnvLogsDeploy, bw.EnvLogsVerbose) {
				log.Printf("initiating periodic healthcheck %f\n", r.Limit())
			}
			tickle.Signal()
		}
	}
}

func MonitorTicklerEvent(l *agent.Peer, d dialers.ContextDialer) MonitorTickler {
	return func(ctx context.Context, tickle *sync.Cond) {
		events := make(chan *agent.Message, 100)
		go agentutil.WatchEvents(ctx, l, d, events)
		for {
			select {
			case m := <-events:
				switch evt := m.Event.(type) {
				case *agent.Message_Deploy:
					switch evt.Deploy.Stage {
					case agent.Deploy_Completed, agent.Deploy_Failed:
						if envx.Boolean(false, bw.EnvLogsDeploy, bw.EnvLogsVerbose) {
							log.Printf("initiating event driven healthcheck %T - %s\n", evt, evt.Deploy.Stage)
						}
						tickle.Signal()
					default:
					}
				default:
				}
			case <-ctx.Done():
				return
			}
		}
	}
}

type Monitor struct {
	ticklers []MonitorTickler
}

func (t Monitor) tickler(ctx context.Context, tickle *sync.Cond, q chan struct{}) {
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

func (t Monitor) Await(ctx context.Context, q chan *pending, d dispatcher, c cluster, check operation, additional ...MonitorTickler) error {
	ctx, done := context.WithCancel(ctx)
	defer done()

	if envx.Boolean(false, bw.EnvLogsDeploy, bw.EnvLogsVerbose) {
		log.Println("event monitoring initiated")
		defer log.Println("event monitoring completed")
	}
	performcheck := make(chan struct{})
	outstanding := make([]*pending, 0, 128)
	tickle := sync.NewCond(&sync.Mutex{})
	checker := newChecker(ctx, check)
	defer checker.Close()

	go t.tickler(ctx, tickle, performcheck)
	for _, tickler := range t.ticklers {
		go tickler(ctx, tickle)
	}

	for _, tickler := range additional {
		go tickler(ctx, tickle)
	}

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
			if envx.Boolean(false, bw.EnvLogsDeploy, bw.EnvLogsVerbose) {
				log.Println("healthchecks", len(outstanding), "/", cap(outstanding), q == nil, "initiated")
			}

			outstanding = checker.run(ctx, outstanding...)

			if envx.Boolean(false, bw.EnvLogsDeploy, bw.EnvLogsVerbose) {
				log.Println("healthchecks", len(outstanding), "/", cap(outstanding), q == nil, "completed")
			}

			if q == nil && len(outstanding) == 0 {
				return nil
			}
		case <-ctx.Done():
			return ctx.Err()
		}
	}
}

func newChecker(ctx context.Context, o operation) checker {
	c := checker{
		q: make(chan *pending),
		r: make(chan *pending),
	}

	for i := 0; i < runtime.NumCPU(); i++ {
		go c.background(ctx, o)
	}

	return c
}

type checker struct {
	q chan *pending
	r chan *pending
}

func (t checker) Close() {
	close(t.q)
}

func (t checker) background(ctx context.Context, o operation) {
	for p := range t.q {
		if err := healthcheck(ctx, p, o); err == nil {
			t.r <- nil
			continue
		}
		t.r <- p
	}
}

func (t checker) run(ctx context.Context, outstanding ...*pending) (remaining []*pending) {
	remaining = make([]*pending, 0, len(outstanding))
	pending := len(outstanding)
	next, outstanding := pop(outstanding)

	for pending > 0 {
		select {
		case <-ctx.Done():
			return
		case t.q <- next:
			if next, outstanding = pop(outstanding); next == nil {
				t.q = nil
			}
		case r := <-t.r:
			if r != nil {
				remaining = append(remaining, r)
			}

			pending--
		}
	}

	return remaining
}

func pop(s []*pending) (fallback *pending, _ []*pending) {
	if len(s) == 0 {
		return fallback, s
	}

	fallback = s[0]

	return fallback, s[1:]
}
