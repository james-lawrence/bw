package deployment

import (
	"fmt"
	"log"
	"sync"
	"sync/atomic"
	"time"

	"github.com/james-lawrence/bw"
	"github.com/james-lawrence/bw/agent"
	"github.com/james-lawrence/bw/agentutil"
	"github.com/james-lawrence/bw/x/logx"
	"github.com/pkg/errors"
)

// partitioner determines the number of nodes to simultaneously deploy to
// based on the total number of nodes.
type partitioner interface {
	Partition(length int) (size int)
}

// applies an operation to the node.
type operation interface {
	Visit(agent.Peer) (agent.Deploy, error)
}

// OperationFunc pure function operation.
type OperationFunc func(agent.Peer) (agent.Deploy, error)

// Visit implements operation.
func (t OperationFunc) Visit(c agent.Peer) (agent.Deploy, error) {
	return t(c)
}

type constantChecker struct {
	Deploy agent.Deploy
}

func (t constantChecker) Visit(agent.Peer) (agent.Deploy, error) {
	return t.Deploy, nil
}

type cluster interface {
	Peers() []agent.Peer
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

// NewDeploy by default deploys operate in one-at-a-time mode.
func NewDeploy(p agent.Peer, di dispatcher, options ...Option) Deploy {
	d := Deploy{
		filter: AlwaysMatch,
		worker: worker{
			c:          make(chan func()),
			wait:       new(sync.WaitGroup),
			check:      constantChecker{Deploy: agent.Deploy{Stage: agent.Deploy_Completed}},
			deploy:     OperationFunc(loggingDeploy),
			dispatcher: di,
			local:      p,
			completed:  new(int64),
			failed:     new(int64),
		},
		partitioner: bw.ConstantPartitioner(1),
	}

	for _, opt := range options {
		opt(&d)
	}

	return d
}

// RunDeploy convience function for executing a deploy.
func RunDeploy(p agent.Peer, c cluster, di dispatcher, options ...Option) (int64, bool) {
	return NewDeploy(p, di, options...).Deploy(c)
}

func loggingDeploy(peer agent.Peer) (agent.Deploy, error) {
	log.Println("deploy triggered for peer", peer.String())
	return agent.Deploy{Stage: agent.Deploy_Deploying}, nil
}

type worker struct {
	c              chan func()
	wait           *sync.WaitGroup
	local          agent.Peer
	dispatcher     dispatcher
	check          operation
	deploy         operation
	filter         Filter
	completed      *int64
	failed         *int64
	ignoreFailures bool
}

func (t worker) work() {
	defer t.wait.Done()
	for f := range t.c {
		// Stop deployment when a single node fails.
		// TODO: make number of failures allowed configurable.
		if atomic.LoadInt64(t.failed) > 0 && !t.ignoreFailures {
			t.dispatcher.Dispatch(agentutil.PeersCompletedEvent(t.local, atomic.AddInt64(t.completed, 1)))
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

func (t worker) DeployTo(peer agent.Peer) {
	t.c <- func() {
		if _, err := t.deploy.Visit(peer); err != nil {
			t.dispatcher.Dispatch(agentutil.LogEvent(t.local, fmt.Sprintf("failed to deploy to: %s - %s\n", peer.Name, err.Error())))
			atomic.AddInt64(t.failed, 1)
			return
		}

		if success := awaitCompletion(t.dispatcher, t.check, peer); !success {
			atomic.AddInt64(t.failed, 1)
		}

		t.dispatcher.Dispatch(agentutil.PeersCompletedEvent(t.local, atomic.AddInt64(t.completed, 1)))
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

	t.Dispatch(agentutil.PeersFoundEvent(t.worker.local, int64(len(nodes))))

	concurrency := t.partitioner.Partition(len(nodes))
	for i := 0; i < concurrency; i++ {
		t.worker.wait.Add(1)
		go t.worker.work()
	}

	t.Dispatch(agentutil.LogEvent(t.worker.local, "waiting for nodes to become ready"))
	awaitCompletion(t, t.worker.check, nodes...)
	t.Dispatch(agentutil.LogEvent(t.worker.local, "nodes are ready, deploying"))

	for _, peer := range nodes {
		t.worker.DeployTo(peer)
	}

	return t.worker.Complete()
}

// Dispatch - implements dispatcher interface.
func (t Deploy) Dispatch(m ...agent.Message) error {
	return logx.MaybeLog(t.worker.dispatcher.Dispatch(m...))
}

// ApplyFilter applies the filter to the set of peers.
func ApplyFilter(s Filter, set ...agent.Peer) []agent.Peer {
	subset := make([]agent.Peer, 0, len(set))
	for _, peer := range set {
		if s.Match(peer) {
			subset = append(subset, peer)
		}
	}

	return subset
}

func awaitCompletion(d dispatcher, check operation, peers ...agent.Peer) bool {
	remaining := make([]agent.Peer, 0, len(peers))
	success := true
	for len(peers) > 0 {
		remaining = remaining[:0]
		for _, peer := range peers {
			deploy, err := check.Visit(peer)
			if err != nil {
				log.Printf("failed to check: %s - %T, %s\n", peer.Name, errors.Cause(err), err)
			} else {
				switch deploy.Stage {
				case agent.Deploy_Completed:
					continue
				case agent.Deploy_Failed:
					success = false
					continue
				}
			}

			remaining = append(remaining, peer)
			time.Sleep(time.Second)
		}

		peers = remaining
	}

	return success
}
