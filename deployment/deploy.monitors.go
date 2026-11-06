package deployment

import (
	"bytes"
	"context"
	"fmt"
	"log"
	"runtime"
	"sync"
	"time"

	"github.com/james-lawrence/bw"
	"github.com/james-lawrence/bw/agent"
	"github.com/james-lawrence/bw/agent/dialers"
	"github.com/james-lawrence/bw/agentutil"
	"github.com/james-lawrence/bw/internal/envx"
	"github.com/james-lawrence/bw/internal/errorsx"
	"github.com/pkg/errors"
	"golang.org/x/time/rate"
)

// monitors monitor a peers deployment to determine success or failures.
type monitor interface {
	// Await monitors the pending nodes until the expected number of reports have been received
	// and every reported node has completed. a nil report is a node that doesn't need monitoring.
	Await(ctx context.Context, q chan *pending, expected int, d dispatcher, c cluster, check operation, additional ...MonitorTickler) error
}

func NewMonitor(ticklers ...MonitorTickler) Monitor {
	return Monitor{
		ticklers: ticklers,
	}
}

type MonitorTickler func(ctx context.Context, tickle *sync.Cond, departed chan *agent.Peer)

func MonitorTicklerPeriodic(d time.Duration) MonitorTickler {
	return MonitorTicklerRate(rate.NewLimiter(rate.Every(d), 1))
}

func MonitorTicklerPeriodicAuto(d time.Duration) MonitorTickler {
	return MonitorTicklerPeriodic(d / 4)
}

func MonitorTicklerRate(r *rate.Limiter) MonitorTickler {
	return func(ctx context.Context, tickle *sync.Cond, departed chan *agent.Peer) {
		defer tickle.Signal()
		for err := r.Wait(ctx); err == nil; err = r.Wait(ctx) {
			if envx.Boolean(false, bw.EnvLogsDeploy, bw.EnvLogsVerbose) {
				log.Printf("initiating periodic healthcheck %f\n", r.Limit())
			}
			tickle.Signal()
		}
	}
}

// MonitorTicklerHeartbeat emits a heartbeat every period while the monitor is running. the context
// ending, either from completion or its deadline, is the expected way for it to stop.
func MonitorTicklerHeartbeat(local *agent.Peer, d dispatcher, period time.Duration) MonitorTickler {
	return func(ctx context.Context, tickle *sync.Cond, departed chan *agent.Peer) {
		t := time.NewTicker(period)
		defer t.Stop()

		for {
			select {
			case <-ctx.Done():
				return
			case <-t.C:
				errorsx.Log(errorsx.Wrap(agentutil.Dispatch(ctx, d, agent.NewDeployHeartbeat(local)), "heartbeat failure"))
			}
		}
	}
}

// MonitorTicklerEvent triggers healthchecks from the events of the cluster. stop is invoked, with the reason,
// when an event ends the deploy of the given deployment coordinated by l (see supersedes).
func MonitorTicklerEvent(l *agent.Peer, d dialers.T, did []byte, stop func(error)) MonitorTickler {
	return func(ctx context.Context, tickle *sync.Cond, departed chan *agent.Peer) {
		events := make(chan *agent.Message, 100)
		go agentutil.WatchEvents(ctx, l, d, events)

		ends := supersedes(l, did)
		tickleOnEvents(ctx, events, tickle, departed, func(dc *agent.DeployCommand) {
			if cause := ends(dc); cause != nil {
				stop(cause)
			}
		})
	}
}

// supersedes returns a function deciding whether a deploy command ends the deploy of the deployment coordinated
// by local. the reason is returned for matches, nil otherwise.
//   - a restart naming the local node as the previous coordinator, a new leader has taken the deploy over.
//   - a cancel, which doesn't identify the deployment: there is a single deploy at a time and a single coordinator.
//   - a done or failed for the deployment.
//
// restarts, done and failed are only matched when the deployment is known.
func supersedes(local *agent.Peer, deploymentID []byte) func(*agent.DeployCommand) error {
	return func(dc *agent.DeployCommand) error {
		same := len(deploymentID) > 0 && bytes.Equal(dc.GetArchive().GetDeploymentID(), deploymentID)

		switch dc.GetCommand() {
		case agent.DeployCommand_Restart:
			if same && dc.GetPrevious().GetName() == local.GetName() {
				return errors.New("deploy restarted by a new coordinator")
			}
		case agent.DeployCommand_Cancel:
			return errors.New("deploy cancelled")
		case agent.DeployCommand_Done, agent.DeployCommand_Failed:
			if same {
				return errors.Errorf("deploy ended elsewhere: %s", dc.GetCommand())
			}
		}

		return nil
	}
}

// tickleOnEvents triggers healthchecks when a node finishes deploying and reports the nodes that
// departed the cluster. returns when the context is done or the events are exhausted. deploy commands
// are passed to onCommand.
func tickleOnEvents(ctx context.Context, events <-chan *agent.Message, tickle *sync.Cond, departed chan *agent.Peer, onCommand func(*agent.DeployCommand)) {
	for {
		select {
		case m, ok := <-events:
			if !ok {
				return
			}

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
			case *agent.Message_Membership:
				if evt.Membership == agent.Message_Departed {
					departed <- m.Peer
				}
			case *agent.Message_DeployCommand:
				onCommand(evt.DeployCommand)
			default:
			}
		case <-ctx.Done():
			return
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

func (t Monitor) Await(ctx context.Context, q chan *pending, expected int, d dispatcher, c cluster, check operation, additional ...MonitorTickler) error {
	if expected <= 0 {
		return nil
	}

	ctx, done := context.WithCancel(ctx)
	defer done()

	if envx.Boolean(false, bw.EnvLogsDeploy, bw.EnvLogsVerbose) {
		log.Println("event monitoring initiated")
		defer log.Println("event monitoring completed")
	}

	departed := make(map[string]*agent.Peer, 128)
	departedchan := make(chan *agent.Peer)
	performcheck := make(chan struct{})
	outstanding := make([]*pending, 0, 128)
	received := 0
	tickle := sync.NewCond(&sync.Mutex{})
	checker := newChecker(ctx, check)
	defer checker.Close()

	go t.tickler(ctx, tickle, performcheck)
	for _, tickler := range t.ticklers {
		go tickler(ctx, tickle, departedchan)
	}

	for _, tickler := range additional {
		go tickler(ctx, tickle, departedchan)
	}

	for {
		select {
		case task, ok := <-q:
			if !ok {
				q = nil
				tickle.Signal()
				continue
			}

			if received++; received >= expected {
				// every node has reported, nothing more will be monitored.
				q = nil
			}

			if task != nil {
				outstanding = append(outstanding, task)
			}

			tickle.Signal()
		case p := <-departedchan:
			departed[p.Name] = p
		case <-performcheck:
			if envx.Boolean(false, bw.EnvLogsDeploy, bw.EnvLogsVerbose) {
				log.Println("healthchecks", len(outstanding), "/", cap(outstanding), q == nil, "initiated")
			}

			outstanding = checker.run(ctx, outstanding...)
			outstanding = dropDeparted(departed, func(p *agent.Peer) {
				// a departed node is not a failed deploy, acknowledge it and stop monitoring it.
				errorsx.Log(agentutil.Dispatch(ctx, d, agent.LogWarning(p, fmt.Sprintf("node %s departed the cluster, no longer waiting on it", p.Ip))))
			}, outstanding...)

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

func dropDeparted(departed map[string]*agent.Peer, acknowledge func(*agent.Peer), outstanding ...*pending) (remaining []*pending) {
	if len(departed) == 0 {
		return outstanding
	}

	remaining = make([]*pending, 0, len(outstanding))
	for _, o := range outstanding {
		if _, ok := departed[o.Name]; ok {
			log.Println("dropping", o.Ip, "node departed")
			acknowledge(o.Peer)
			close(o.done)
			continue
		}

		remaining = append(remaining, o)
	}

	return remaining
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
