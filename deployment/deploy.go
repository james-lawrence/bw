package deployment

import (
	"log"
	"math"
	"sync"
	"sync/atomic"
	"time"

	"github.com/hashicorp/memberlist"
)

type Stage int

const (
	// StageAwaitingForReady - deployer is waiting for all nodes to be ready for deployment.
	StageWaitingForReady Stage = iota
	// StageDeploying - deployer is deploying.
	StageDeploying
	//StageDone - deployer has finished.
	StageDone
)

type NodeUpdate struct {
	Peer   *memberlist.Node
	Status error
}

func NewEvents() *Events {
	return &Events{
		NodesFound:     make(chan int64),
		NodesCompleted: make(chan int64),
		Status:         make(chan NodeUpdate, 10),
		StageUpdate:    make(chan Stage),
	}
}

type Events struct {
	NodesFound     chan int64
	NodesCompleted chan int64
	completed      int64
	StageUpdate    chan Stage
	Status         chan NodeUpdate
}

// PercentagePartitioner size is based on the percentage. has an upper bound of 1.0.
type PercentagePartitioner float64

func (t PercentagePartitioner) Partition(length int) int {
	ratio := math.Min(float64(t), 1.0)
	// log.Println("length", length, "ratio", ratio)
	computed := int(math.Max(math.Floor(float64(length)*ratio), 1.0))
	// log.Println("computed", computed)
	return computed
}

// ConstantPartitioner partition will return the specified min(length, size).
type ConstantPartitioner int

// Partition implements partitioner
func (t ConstantPartitioner) Partition(length int) int {
	return max(1, min(length, int(t)))
}

// partitioner determines the number of nodes to simultaneously deploy to
// based on the total number of nodes.
type partitioner interface {
	Partition(length int) (size int)
}

// checker checks the current status of a node.
type checker interface {
	Check(*memberlist.Node) error
}

type constantChecker struct {
	Status
}

func (t constantChecker) Check(*memberlist.Node) error {
	return t.Status
}

type cluster interface {
	Members() []*memberlist.Node
}

type Option func(*Deploy)

func DeployOptionFilter(x Filter) Option {
	return func(d *Deploy) {
		d.filter = x
	}
}

func DeployOptionPartitioner(x partitioner) Option {
	return func(d *Deploy) {
		d.partitioner = x
	}
}

func DeployOptionChecker(x checker) Option {
	return func(d *Deploy) {
		d.checker = x
	}
}

func DeployOptionDeployer(deployer func(peer *memberlist.Node) error) Option {
	return func(d *Deploy) {
		d.worker.deployer = deployer
	}
}

// NewDeploy by default deploys operate in one-at-a-time mode.
func NewDeploy(h *Events, options ...Option) Deploy {
	d := Deploy{
		filter: AlwaysMatch,
		worker: worker{
			c:        make(chan func()),
			wait:     &sync.WaitGroup{},
			checker:  constantChecker{Status: ready{}},
			deployer: loggingDeployer,
			handler:  h,
		},
		partitioner: ConstantPartitioner(1),
	}

	for _, opt := range options {
		opt(&d)
	}

	return d
}

func loggingDeployer(peer *memberlist.Node) error {
	log.Println("deploy triggered for peer", peer.String())
	return nil
}

type worker struct {
	c    chan func()
	wait *sync.WaitGroup
	checker
	filter   Filter
	deployer func(peer *memberlist.Node) error
	handler  *Events
}

func (t worker) work() {
	defer t.wait.Done()
	for f := range t.c {
		f()
	}
}

func (t worker) Complete() {
	close(t.c)
	t.wait.Wait()
}

func (t worker) DeployTo(peer *memberlist.Node) {
	t.c <- func() {
		if err := t.deployer(peer); err != nil {
			log.Printf("failed to deploy to: %s - %+v\n", peer.Name, err)
			return
		}
		awaitCompletion(t.handler, t.checker, peer)
		log.Println("emitting completion")
		t.handler.NodesCompleted <- atomic.AddInt64(&t.handler.completed, 1)
		log.Println("emitted completion")
	}
}

type Deploy struct {
	filter Filter
	partitioner
	worker
}

func (t Deploy) Deploy(c cluster) {
	nodes := _filter(c.Members(), t.filter)
	t.worker.handler.NodesFound <- int64(len(nodes))

	for i := 0; i < t.partitioner.Partition(len(nodes)); i++ {
		t.worker.wait.Add(1)
		go t.worker.work()
	}

	t.worker.handler.StageUpdate <- StageWaitingForReady
	awaitCompletion(t.worker.handler, t.worker.checker, nodes...)

	t.worker.handler.StageUpdate <- StageDeploying
	for _, peer := range nodes {
		t.worker.DeployTo(peer)
	}

	t.worker.Complete()
	t.worker.handler.StageUpdate <- StageDone
}

func _filter(set []*memberlist.Node, s Filter) []*memberlist.Node {
	subset := make([]*memberlist.Node, 0, len(set))
	for _, peer := range set {
		if s.Match(peer) {
			subset = append(subset, peer)
		}
	}

	return subset
}

func awaitCompletion(e *Events, c checker, nodes ...*memberlist.Node) {
	remaining := make([]*memberlist.Node, 0, len(nodes))
	for len(nodes) > 0 {
		remaining = remaining[:0]
		for _, peer := range nodes {
			s := c.Check(peer)

			e.Status <- NodeUpdate{Status: s, Peer: peer}

			if IsReady(s) {
				continue
			}

			if IsFailed(s) {
				continue
			}

			remaining = append(remaining, peer)
			time.Sleep(time.Second)
		}

		nodes = remaining
	}
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}

func max(a, b int) int {
	if a < b {
		return b
	}
	return a
}
