package deployment

import (
	"context"
	"log"
	"sync"
	"sync/atomic"
	"time"

	"github.com/james-lawrence/bw"
	"github.com/james-lawrence/bw/agent"
	"github.com/james-lawrence/bw/agentutil"
	"github.com/james-lawrence/bw/internal/envx"
	"github.com/james-lawrence/bw/internal/errorsx"
	"github.com/james-lawrence/bw/internal/timex"
	"github.com/pkg/errors"
	"golang.org/x/time/rate"
)

const deployGracePeriod = time.Minute

// partitioner determines the number of nodes to simultaneously deploy to
// based on the total number of nodes.
type partitioner interface {
	Partition(length int) (size int)
}

// applies an operation to the node.
type operation interface {
	Visit(context.Context, *agent.Peer) (*agent.Deploy, error)
}

// OperationFunc pure function operation.
type OperationFunc func(context.Context, *agent.Peer) (*agent.Deploy, error)

// Visit implements operation.
func (t OperationFunc) Visit(ctx context.Context, c *agent.Peer) (*agent.Deploy, error) {
	return t(ctx, c)
}

type constantChecker struct {
	Deploy *agent.Deploy
}

func (t constantChecker) Visit(context.Context, *agent.Peer) (*agent.Deploy, error) {
	return t.Deploy, nil
}

type cluster interface {
	Peers() []*agent.Peer
}

// Option ...
type Option func(*Deploy)

// DeployOptionFilter filter nodes to deploy to.
func DeployOptionFilter(x Filter) Option {
	return func(d *Deploy) {
		d.filter = x
	}
}

// DeployOptionPartitioner set the strategy for partitioning the cluster into sets.
func DeployOptionPartitioner(x partitioner) Option {
	return func(d *Deploy) {
		d.partitioner = x
	}
}

// DeployOptionChecker set the strategy for checking the state of a node.
func DeployOptionChecker(x operation) Option {
	return func(d *Deploy) {
		d.check = x
	}
}

// DeployOptionDeployer set the strategy for deploying.
func DeployOptionDeployer(o operation) Option {
	return func(d *Deploy) {
		d.worker.deploy = o
	}
}

// DeployOptionMonitor set the monitoring strategy
// for a deployment defaults to a polling strategy.
func DeployOptionMonitor(m monitor) Option {
	return func(d *Deploy) {
		d.monitor = m
	}
}

// DeployOptionIgnoreFailures set whether or not to ignore failures.
func DeployOptionIgnoreFailures(ignore bool) Option {
	return func(d *Deploy) {
		d.worker.ignoreFailures = ignore
	}
}

// DeployOptionTimeoutGrace set the timeout for each deployment. if it takes longer than
// the provided timeout + a small grace period then give up and consider it failed.
func DeployOptionTimeoutGrace(t time.Duration) Option {
	return func(d *Deploy) {
		d.worker.timeout = t + deployGracePeriod
	}
}

// DeployOptionTimeout set the timeout for each deployment. if it takes longer than
// the provided timeout + a small grace period then give up and consider it failed.
func DeployOptionTimeout(t time.Duration) Option {
	return func(d *Deploy) {
		d.worker.timeout = t
	}
}

// DeployOptionHeartbeatFrequency frequency at which to emit heartbeat events.
func DeployOptionHeartbeatFrequency(t time.Duration) Option {
	return func(d *Deploy) {
		d.worker.heartbeat = timex.DurationOrDefault(t, 5*time.Second)
	}
}

// NewDeploy by default deploys operate in one-at-a-time mode.
func NewDeploy(p *agent.Peer, di dispatcher, options ...Option) Deploy {
	d := Deploy{
		filter: AlwaysMatch,
		worker: worker{
			c:          make(chan func(context.Context) error),
			wait:       new(sync.WaitGroup),
			check:      constantChecker{Deploy: &agent.Deploy{Stage: agent.Deploy_Completed}},
			deploy:     OperationFunc(loggingDeploy),
			dispatcher: di,
			local:      p,
			completed:  new(int64),
			failed:     new(int64),
			timeout:    bw.DefaultDeployTimeout + deployGracePeriod,
			heartbeat:  5 * time.Second,
			queue:      make(chan *pending),
		},
		partitioner: bw.ConstantPartitioner(1),
	}

	for _, opt := range options {
		opt(&d)
	}

	if d.monitor == nil {
		d.monitor = NewMonitor(MonitorTicklerPeriodicAuto(d.timeout))
	}

	return d
}

// RunDeploy convience function for executing a deploy.
func RunDeploy(p *agent.Peer, c cluster, di dispatcher, options ...Option) (int64, bool) {
	return NewDeploy(p, di, options...).Deploy(c)
}

func loggingDeploy(ctx context.Context, peer *agent.Peer) (*agent.Deploy, error) {
	log.Println("deploy triggered for peer", peer.String())
	return &agent.Deploy{Stage: agent.Deploy_Deploying}, nil
}

type pending struct {
	*agent.Peer
	timeout time.Duration
	done    chan error
}

func newPending(p *agent.Peer, d time.Duration) *pending {
	return &pending{Peer: p, timeout: d, done: make(chan error, 1)}
}

type worker struct {
	c              chan func(context.Context) error
	wait           *sync.WaitGroup
	local          *agent.Peer
	dispatcher     dispatcher
	monitor        monitor
	check          operation
	deploy         operation
	completed      *int64
	failed         *int64
	ignoreFailures bool
	timeout        time.Duration
	heartbeat      time.Duration
	queue          chan *pending
}

func (t worker) work(ctx context.Context) {
	defer t.wait.Done()
	for op := range t.c {
		// Stop deployment when a single node fails.
		if atomic.LoadInt64(t.failed) > 0 && !t.ignoreFailures {
			errorsx.Log(agentutil.Dispatch(ctx, t.dispatcher, agent.PeersCompletedEvent(t.local, atomic.AddInt64(t.completed, 1))))
			continue
		}

		if err := op(ctx); err != nil {
			log.Println(err)
			atomic.AddInt64(t.failed, 1)
			errorsx.Log(agentutil.ReliableDispatch(ctx, t.dispatcher, agent.LogError(t.local, err)))
		} else {
			errorsx.Log(agentutil.ReliableDispatch(ctx, t.dispatcher, agent.PeersCompletedEvent(t.local, atomic.AddInt64(t.completed, 1))))
		}
	}
}

func (t worker) Complete() (int64, bool) {
	t.wait.Wait()
	failures := atomic.LoadInt64(t.failed)
	return failures, t.ignoreFailures || failures == 0
}

func (t worker) DeployTo(ctx context.Context, peer *agent.Peer) error {
	task := newPending(peer, t.timeout)
	perform := func(deadline context.Context) error {
		if envx.Boolean(false, bw.EnvLogsDeploy, bw.EnvLogsVerbose) {
			log.Println("deploy to", peer.Ip, "initiated")
			defer log.Println("deploy to", peer.Ip, "completed")
		}

		if _, err := t.deploy.Visit(deadline, peer); err != nil {
			return errors.Wrapf(err, "failed to deploy to: %s", peer.Ip)
		}

		// send the task to be monitored.
		select {
		case t.queue <- task:
		case <-deadline.Done():
			return errors.Wrapf(deadline.Err(), "failed to deploy to: %s", peer.Ip)
		}

		select {
		case <-deadline.Done():
			return errors.Wrapf(deadline.Err(), "failed to deploy to: %s", peer.Ip)
		case cause := <-task.done:
			return errors.Wrapf(cause, "failed to deploy to: %s", peer.Ip)
		}
	}

	select {
	case t.c <- perform:
	case <-ctx.Done():
		return ctx.Err()
	}

	return nil
}

// Deploy - handles a deployment.
type Deploy struct {
	filter Filter
	partitioner
	worker
}

// Deploy deploy to the cluster. returns deployment results.
// failed nodes and if it was considered a success.
func (t Deploy) Deploy(c cluster) (int64, bool) {
	ctx, done := context.WithTimeout(context.Background(), t.worker.timeout+deployGracePeriod)
	defer done()

	nodes := ApplyFilter(t.filter, c.Peers()...)
	errorsx.Log(agentutil.Dispatch(ctx, t.dispatcher, agent.PeersFoundEvent(t.worker.local, int64(len(nodes)))))

	concurrency := t.partitioner.Partition(len(nodes))
	for i := 0; i < concurrency; i++ {
		t.worker.wait.Add(1)
		go t.worker.work(ctx)
	}

	initial := make(chan *pending, len(nodes))
	for _, n := range nodes {
		initial <- newPending(n, t.timeout)
	}
	close(initial)

	go heartbeat(ctx, t.worker.local, rate.Every(t.worker.heartbeat), t.dispatcher)

	if failure := t.monitor.Await(ctx, initial, t.dispatcher, c, t.worker.check); failure != nil {
		switch errors.Cause(failure).(type) {
		case errorsx.Timeout:
			errorsx.Log(
				agentutil.Dispatch(
					ctx,
					t.dispatcher,
					agent.LogEvent(t.worker.local, "timed out while waiting for nodes to complete, maybe try cancelling the current deploy"),
				),
			)
			return 0, false
		default:
		}
	}

	errorsx.Log(agentutil.Dispatch(ctx, t.dispatcher, agent.LogEvent(t.worker.local, "nodes are ready, deploying")))

	go func() {
		for _, peer := range nodes {
			errorsx.Log(t.worker.DeployTo(ctx, peer))
		}

		close(t.c)
		t.wait.Wait()
		close(t.queue)
	}()

	failure := errorsx.Ignore(
		t.worker.monitor.Await(ctx, t.queue, t.dispatcher, c, t.worker.check),
		context.Canceled,
	)
	if failure != nil {
		switch errors.Cause(failure).(type) {
		case errorsx.Timeout:
			errorsx.Log(
				agentutil.Dispatch(ctx, t.dispatcher, agent.LogEvent(t.worker.local, "timed out while waiting for nodes to complete")),
			)
		default:
		}
	}

	return t.worker.Complete()
}

// ApplyFilter applies the filter to the set of peers.
func ApplyFilter(s Filter, set ...*agent.Peer) []*agent.Peer {
	subset := make([]*agent.Peer, 0, len(set))
	for _, peer := range set {
		if s.Match(peer) {
			subset = append(subset, peer)
		}
	}

	return subset
}

func healthcheck(ctx context.Context, task *pending, op operation) (err error) {
	var (
		deploy *agent.Deploy
	)

	if envx.Boolean(false, bw.EnvLogsDeploy, bw.EnvLogsVerbose) {
		log.Println("healthcheck", task.Peer.Ip, task.timeout, "initiated")
		defer log.Println("healthcheck", task.Peer.Ip, task.timeout, "completed")
	}

	ctx, done := context.WithTimeout(ctx, task.timeout)
	defer done()

	if deploy, err = op.Visit(ctx, task.Peer); err != nil {
		return err
	}

	did := bw.RandomID("unknown deployment")
	if deploy.Archive != nil {
		did = bw.RandomID(deploy.Archive.DeploymentID)
	}

	switch deploy.Stage {
	case agent.Deploy_Completed:
		close(task.done)
		return nil
	case agent.Deploy_Failed:
		task.done <- errors.Errorf("%s: deployment has failed", did)
		close(task.done)
		return nil
	default:
		return errors.New(deploy.Stage.String())
	}
}
