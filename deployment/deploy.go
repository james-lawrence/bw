package deployment

import (
	"context"
	"fmt"
	"log"
	"math"
	"sync/atomic"
	"time"

	"github.com/james-lawrence/bw"
	"github.com/james-lawrence/bw/agent"
	"github.com/james-lawrence/bw/agentutil"
	"github.com/james-lawrence/bw/internal/asynccompute"
	"github.com/james-lawrence/bw/internal/envx"
	"github.com/james-lawrence/bw/internal/errorsx"
	"github.com/james-lawrence/bw/internal/timex"
	"github.com/pkg/errors"
)

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

// DeployOptionAttempts set the number of attempts a single node is given before it is considered failed.
// values less than 1 are treated as 1.
func DeployOptionAttempts(n int64) Option {
	return func(d *Deploy) {
		d.worker.attempts = max(n, 1)
	}
}

// DeployOptionTimeout set the timeout for the entire workload. if the deploy takes longer than
// the provided timeout + the grace period then give up and consider it failed.
func DeployOptionTimeout(t time.Duration) Option {
	return func(d *Deploy) {
		d.worker.timeout = t
	}
}

// DeployOptionGracePeriod set the grace period added to the timeout of the entire workload.
func DeployOptionGracePeriod(t time.Duration) Option {
	return func(d *Deploy) {
		d.worker.graceperiod = t
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
			check:       constantChecker{Deploy: &agent.Deploy{Stage: agent.Deploy_Completed}},
			deploy:      OperationFunc(loggingDeploy),
			dispatcher:  di,
			local:       p,
			completed:   new(int64),
			failed:      new(int64),
			attempts:    1,
			timeout:     bw.DefaultDeployTimeout,
			graceperiod: time.Minute,
			heartbeat:   5 * time.Second,
			queue:       make(chan *pending),
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
func RunDeploy(ctx context.Context, p *agent.Peer, c cluster, di dispatcher, options ...Option) (int64, bool) {
	return NewDeploy(p, di, options...).Deploy(ctx, c)
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
	local          *agent.Peer
	dispatcher     dispatcher
	monitor        monitor
	check          operation
	deploy         operation
	completed      *int64
	failed         *int64
	ignoreFailures bool
	attempts       int64
	timeout        time.Duration
	graceperiod    time.Duration
	heartbeat      time.Duration
	queue          chan *pending
}

// work deploys to the peer and records the result, it never fails the pool. failures are tracked
// by the worker so they can be reported as a count.
func (t worker) work(ctx context.Context, peer *agent.Peer) error {
	// Stop deployment when a single node fails.
	if atomic.LoadInt64(t.failed) > 0 && !t.ignoreFailures {
		t.report(ctx, nil)
		errorsx.Log(agentutil.ReliableDispatch(ctx, t.dispatcher, agent.PeersCompletedEvent(t.local, atomic.AddInt64(t.completed, 1))))
		return nil
	}

	if err := t.deployTo(ctx, peer); err != nil {
		log.Println(err)
		atomic.AddInt64(t.failed, 1)
		errorsx.Log(agentutil.ReliableDispatch(ctx, t.dispatcher, agent.LogError(t.local, err)))
	} else {
		errorsx.Log(agentutil.ReliableDispatch(ctx, t.dispatcher, agent.PeersCompletedEvent(t.local, atomic.AddInt64(t.completed, 1))))
	}

	return nil
}

// Complete reports the number of failed nodes and whether the deploy is considered a success.
// a deploy whose context has ended is never a success, nodes may not have been attempted.
func (t worker) Complete(ctx context.Context) (int64, bool) {
	failures := atomic.LoadInt64(t.failed)
	if cause := ctx.Err(); cause != nil {
		errorsx.Log(errorsx.Wrap(cause, "deploy did not complete"))
		return failures, false
	}

	return failures, t.ignoreFailures || failures == 0
}

// report the node to the monitor, every node reports exactly once so the monitor knows when the
// workload is complete. a nil task is a node that doesn't need to be monitored. returns false
// if the context ended before the report was delivered.
func (t worker) report(ctx context.Context, task *pending) bool {
	select {
	case t.queue <- task:
		return true
	case <-ctx.Done():
		return false
	}
}

func (t worker) deployTo(ctx context.Context, peer *agent.Peer) error {
	task := newPending(peer, t.timeout)

	if envx.Boolean(false, bw.EnvLogsDeploy, bw.EnvLogsVerbose) {
		log.Println("deploy to", peer.Ip, "initiated")
		defer log.Println("deploy to", peer.Ip, "completed")
	}

	// only dispatching the deploy is retried. once the task has been reported the monitor owns the
	// node, and it expects exactly one report per node.
	var err error
	for attempt := int64(1); attempt <= max(t.attempts, 1); attempt++ {
		if _, err = t.deploy.Visit(ctx, peer); err == nil {
			break
		}

		if attempt < t.attempts {
			log.Printf("deploy to %s failed, attempt %d/%d: %v\n", peer.Ip, attempt, t.attempts, err)
			select {
			case <-ctx.Done():
				t.report(ctx, nil)
				return errors.Wrapf(ctx.Err(), "failed to deploy to: %s", peer.Ip)
			case <-time.After(time.Second):
			}
		}
	}

	if err != nil {
		t.report(ctx, nil)
		return errors.Wrapf(err, "failed to deploy to: %s", peer.Ip)
	}

	// send the task to be monitored.
	if !t.report(ctx, task) {
		return errors.Wrapf(ctx.Err(), "failed to deploy to: %s", peer.Ip)
	}

	select {
	case <-ctx.Done():
		return errors.Wrapf(ctx.Err(), "failed to deploy to: %s", peer.Ip)
	case cause := <-task.done:
		return errors.Wrapf(cause, "failed to deploy to: %s", peer.Ip)
	}
}

// Deploy - handles a deployment.
type Deploy struct {
	filter Filter
	partitioner
	worker
}

// Deploy deploy to the cluster. returns deployment results.
// failed nodes and if it was considered a success. cancelling the context stops the deploy,
// which is then never a success.
func (t Deploy) Deploy(ctx context.Context, c cluster) (int64, bool) {
	ctx, done := context.WithTimeout(ctx, t.worker.timeout+t.worker.graceperiod)
	defer done()

	nodes := ApplyFilter(t.filter, c.Peers()...)
	errorsx.Log(agentutil.Dispatch(ctx, t.dispatcher, agent.PeersFoundEvent(t.worker.local, int64(len(nodes)))))

	initial := make(chan *pending, len(nodes))
	for _, n := range nodes {
		initial <- newPending(n, t.timeout)
	}
	close(initial)

	heartbeat := MonitorTicklerHeartbeat(t.worker.local, t.dispatcher, t.worker.heartbeat)

	if failure := t.monitor.Await(ctx, initial, len(nodes), t.dispatcher, c, t.worker.check, heartbeat); errorsx.Is(failure, context.DeadlineExceeded) {
		// the deploy's context has ended, the notification must not depend on it.
		errorsx.Log(
			agentutil.Dispatch(
				context.WithoutCancel(ctx),
				t.dispatcher,
				agent.LogWarning(t.worker.local, "timed out while waiting for nodes to complete, maybe try cancelling the current deploy"),
			),
		)
		return 0, false
	} else if errorsx.Is(failure, context.Canceled) {
		t.stopped(ctx)
		return 0, false
	}

	errorsx.Log(agentutil.Dispatch(ctx, t.dispatcher, agent.LogUser(t.worker.local, "nodes are ready, deploying")))

	concurrency := min(max(t.partitioner.Partition(len(nodes)), 1), math.MaxUint16)
	pool := asynccompute.New(
		t.worker.work,
		asynccompute.Workers[*agent.Peer](uint16(concurrency)),
		// every node fits in the backlog so queueing never blocks on the workers.
		asynccompute.Backlog[*agent.Peer](uint16(min(len(nodes), math.MaxUint16))),
	)

	for _, peer := range nodes {
		errorsx.Log(pool.Run(ctx, peer))
	}

	failure := t.worker.monitor.Await(ctx, t.queue, len(nodes), t.dispatcher, c, t.worker.check, heartbeat)

	if errorsx.Is(failure, context.DeadlineExceeded) {
		errorsx.Log(
			agentutil.Dispatch(context.WithoutCancel(ctx), t.dispatcher, agent.LogWarning(t.worker.local, "timed out while waiting for nodes to complete")),
		)
	}

	if errorsx.Is(failure, context.Canceled) {
		t.stopped(ctx)
	}

	// the workers must be done before reporting, the monitor can return early when the context ends.
	errorsx.Log(asynccompute.Shutdown(ctx, pool))

	return t.worker.Complete(ctx)
}

// stopped reports why the deploy was stopped before it completed, the cause of the cancelled context.
// the notification must not depend on the context that has ended.
func (t Deploy) stopped(ctx context.Context) {
	cause := context.Cause(ctx)
	log.Println("deploy stopped:", cause)
	errorsx.Log(
		agentutil.Dispatch(
			context.WithoutCancel(ctx),
			t.dispatcher,
			agent.LogWarning(t.worker.local, fmt.Sprintf("deploy stopped by the coordinator: %v", cause)),
		),
	)
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
		did    bw.RandomID
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
