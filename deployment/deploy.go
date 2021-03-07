package deployment

import (
	"context"
	"fmt"
	"log"
	"sync"
	"sync/atomic"
	"time"

	"github.com/james-lawrence/bw"
	"github.com/james-lawrence/bw/agent"
	"github.com/james-lawrence/bw/agentutil"
	"github.com/james-lawrence/bw/backoff"
	"github.com/james-lawrence/bw/internal/x/errorsx"
	"github.com/james-lawrence/bw/internal/x/timex"
	"github.com/pkg/errors"
)

const deployGracePeriod = time.Minute

// partitioner determines the number of nodes to simultaneously deploy to
// based on the total number of nodes.
type partitioner interface {
	Partition(length int) (size int)
}

// applies an operation to the node.
type operation interface {
	Visit(*agent.Peer) (*agent.Deploy, error)
}

// OperationFunc pure function operation.
type OperationFunc func(*agent.Peer) (*agent.Deploy, error)

// Visit implements operation.
func (t OperationFunc) Visit(c *agent.Peer) (*agent.Deploy, error) {
	return t(c)
}

type constantChecker struct {
	Deploy *agent.Deploy
}

func (t constantChecker) Visit(*agent.Peer) (*agent.Deploy, error) {
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

// DeployOptionIgnoreFailures set whether or not to ignore failures.
func DeployOptionIgnoreFailures(ignore bool) Option {
	return func(d *Deploy) {
		d.worker.ignoreFailures = ignore
	}
}

// DeployOptionTimeout set the timeout for each deployment. if it takes longer than
// the provided timeout + a small grace period then give up and consider it failed.
func DeployOptionTimeout(t time.Duration) Option {
	return func(d *Deploy) {
		d.worker.timeout = t + deployGracePeriod
	}
}

// NewDeploy by default deploys operate in one-at-a-time mode.
func NewDeploy(p *agent.Peer, di dispatcher, options ...Option) Deploy {
	d := Deploy{
		filter: AlwaysMatch,
		worker: worker{
			c:          make(chan func()),
			wait:       new(sync.WaitGroup),
			check:      constantChecker{Deploy: &agent.Deploy{Stage: agent.Deploy_Completed}},
			deploy:     OperationFunc(loggingDeploy),
			dispatcher: di,
			local:      p,
			completed:  new(int64),
			failed:     new(int64),
			timeout:    bw.DefaultDeployTimeout + deployGracePeriod,
		},
		partitioner: bw.ConstantPartitioner(1),
	}

	for _, opt := range options {
		opt(&d)
	}

	return d
}

// RunDeploy convience function for executing a deploy.
func RunDeploy(p *agent.Peer, c cluster, di dispatcher, options ...Option) (int64, bool) {
	return NewDeploy(p, di, options...).Deploy(c)
}

func loggingDeploy(peer *agent.Peer) (*agent.Deploy, error) {
	log.Println("deploy triggered for peer", peer.String())
	return &agent.Deploy{Stage: agent.Deploy_Deploying}, nil
}

type worker struct {
	c              chan func()
	wait           *sync.WaitGroup
	local          *agent.Peer
	dispatcher     dispatcher
	check          operation
	deploy         operation
	filter         Filter
	completed      *int64
	failed         *int64
	ignoreFailures bool
	timeout        time.Duration
}

func (t worker) work() {
	defer t.wait.Done()
	for f := range t.c {
		// Stop deployment when a single node fails.
		// TODO: make number of failures allowed configurable.
		if atomic.LoadInt64(t.failed) > 0 && !t.ignoreFailures {
			agentutil.Dispatch(t.dispatcher, agentutil.PeersCompletedEvent(t.local, atomic.AddInt64(t.completed, 1)))
			continue
		}

		f()
	}
}

func (t worker) Complete() (int64, bool) {
	close(t.c)
	t.wait.Wait()
	failures := atomic.LoadInt64(t.failed)
	return failures, t.ignoreFailures || failures == 0
}

func (t worker) DeployTo(peer *agent.Peer) {
	t.c <- func() {
		deadline, done := context.WithTimeout(context.Background(), t.timeout)
		defer done()

		if _, err := t.deploy.Visit(peer); err != nil {
			agentutil.ReliableDispatch(deadline, t.dispatcher, agentutil.LogEvent(t.local, fmt.Sprintf("failed to deploy to: %s - %s\n", peer.Name, err.Error())))
			atomic.AddInt64(t.failed, 1)
			return
		}

		if failure := awaitCompletion(t.timeout, t.dispatcher, t.check, nil, peer); failure != nil {
			atomic.AddInt64(t.failed, 1)
			switch errors.Cause(failure).(type) {
			case errorsx.Timeout:
				agentutil.ReliableDispatch(deadline, t.dispatcher, agentutil.LogEvent(t.local, fmt.Sprintf("failed to deploy to: %s - %s\n", peer.Name, failure)))
			default:
			}
		}

		agentutil.ReliableDispatch(deadline, t.dispatcher, agentutil.PeersCompletedEvent(t.local, atomic.AddInt64(t.completed, 1)))
	}
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
	nodes := ApplyFilter(t.filter, c.Peers()...)
	agentutil.Dispatch(t.dispatcher, agentutil.PeersFoundEvent(t.worker.local, int64(len(nodes))))

	concurrency := t.partitioner.Partition(len(nodes))
	for i := 0; i < concurrency; i++ {
		t.worker.wait.Add(1)
		go t.worker.work()
	}

	agentutil.Dispatch(t.dispatcher, agentutil.LogEvent(t.worker.local, "waiting for nodes to become ready"))
	if failure := awaitCompletion(t.worker.timeout, t.dispatcher, t.worker.check, c, nodes...); failure != nil {
		switch errors.Cause(failure).(type) {
		case errorsx.Timeout:
			agentutil.Dispatch(t.dispatcher, agentutil.LogEvent(t.worker.local, "timed out while waiting for nodes to complete, maybe try cancelling the current deploy"))
			return 0, false
		default:
		}
	}

	agentutil.Dispatch(t.dispatcher, agentutil.LogEvent(t.worker.local, "nodes are ready, deploying"))

	for _, peer := range nodes {
		t.worker.DeployTo(peer)
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

func awaitCompletion(timeout time.Duration, d dispatcher, check operation, c cluster, peers ...*agent.Peer) error {
	remaining := make([]*agent.Peer, 0, len(peers))
	failed := error(nil)

	b := backoff.Maximum(timex.DurationMin(time.Minute, timeout/4), backoff.Exponential(time.Second))
	deadline := time.Now().Add(timeout)

	for attempt := 0; len(peers) > 0; attempt++ {
		remaining = remaining[:0]
		for _, peer := range peers {
			// if deadline has passed, give up.
			if time.Now().After(deadline) {
				failed = errorsx.Compact(failed, errorsx.Timedout(errors.Errorf("%s: deadline passed for deploy", peer.Name), timeout))
				continue
			}

			if deploy, err := check.Visit(peer); err == nil {
				switch deploy.Stage {
				case agent.Deploy_Completed:
					continue
				case agent.Deploy_Failed:
					did := bw.RandomID("unknown deployment")
					if deploy.Archive != nil {
						did = bw.RandomID(deploy.Archive.DeploymentID)
					}
					failed = errorsx.Compact(failed, errors.Errorf("%s: deployment has failed", did))
					continue
				}
			} else {
				log.Printf("failed to check: %s - %T, %s\n", peer.Name, errors.Cause(err), err)
			}

			remaining = append(remaining, peer)
		}

		// checking if the failed nodes are still within the cluster.
		if c != nil {
			remaining = ApplyFilter(Peers(c.Peers()...), remaining...)
		}

		if len(remaining) > 0 {
			if d := b.Backoff(attempt); time.Now().Add(d).Before(deadline) {
				log.Println("sleeping before next attempt", d, deadline)
				time.Sleep(d)
			} else {
				time.Sleep(deadline.Sub(time.Now()))
			}
		}

		peers = remaining
	}

	return failed
}
