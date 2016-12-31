package deployment

import (
	"log"
	"math"
	"time"

	"github.com/hashicorp/memberlist"
)

// PercentagePartitioner size is based on the percentage. has an upper bound of 1.0.
type PercentagePartitioner float64

func (t PercentagePartitioner) Partition(length int) int {
	ratio := math.Min(float64(t), 1.0)
	log.Println("length", length, "ratio", ratio)
	computed := int(math.Max(math.Floor(float64(length)*ratio), 1.0))
	log.Println("computed", computed)
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

// filter determines if a node should be deployed to based on some conditions.
type filter interface {
	Match(*memberlist.Node) bool
}

type all bool

func (t all) Match(*memberlist.Node) bool {
	return bool(t)
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

func DeployerOptionFilter(x filter) Option {
	return func(d *Deploy) {
		d.filter = x
	}
}

func DeployerOptionPartitioner(x partitioner) Option {
	return func(d *Deploy) {
		d.partitioner = x
	}
}

func DeployerOptionChecker(x checker) Option {
	return func(d *Deploy) {
		d.checker = x
	}
}

func NewDeploy(deployer func(peer *memberlist.Node) error, options ...Option) Deploy {
	d := Deploy{
		filter:      all(true),
		partitioner: PercentagePartitioner(0.50),
		checker:     constantChecker{Status: ready{}},
		deployer:    deployer,
	}

	for _, opt := range options {
		opt(&d)
	}

	return d
}

type Deploy struct {
	filter
	partitioner
	checker
	deployer func(peer *memberlist.Node) error
}

func (t Deploy) Deploy(c cluster) {
	nodes := _filter(c.Members(), t.filter)
	log.Println("waiting for cluster to enter ready state")
	awaitCompletion(t.checker, nodes)
	log.Println("everything is gtg, deploying")
	for _, partition := range partition(len(nodes), t.partitioner.Partition(len(nodes))) {
		subset := nodes[partition.Min:partition.Max]
		deployTo(t.deployer, subset...)
		log.Println("awaiting completion")
		awaitCompletion(t.checker, subset)
	}

	log.Println("completed")
}

func _filter(set []*memberlist.Node, s filter) []*memberlist.Node {
	subset := make([]*memberlist.Node, 0, len(set))
	for _, peer := range set {
		if s.Match(peer) {
			subset = append(subset, peer)
		}
	}

	return subset
}

func partition(length, partitionSize int) []struct{ Min, Max int } {
	var (
		i int
	)

	if length == 1 {
		return []struct{ Min, Max int }{{Min: 0, Max: 1}}
	}

	numFullPartitions, leftOver := int(length/partitionSize), length%partitionSize
	partitions := make([]struct{ Min, Max int }, 0, numFullPartitions+1)
	for ; i < numFullPartitions; i++ {
		partitions = append(partitions, struct{ Min, Max int }{Min: i * partitionSize, Max: (i + 1) * partitionSize})
	}

	if leftOver != 0 { // left over
		partitions = append(partitions, struct{ Min, Max int }{Min: i * partitionSize, Max: length})
	}

	return partitions
}

func deployTo(deployer func(peer *memberlist.Node) error, nodes ...*memberlist.Node) error {
	for _, peer := range nodes {
		if err := deployer(peer); err != nil {
			log.Println("failed to deploy to", peer.Name, err)
		}
	}

	return nil
}

func awaitCompletion(c checker, nodes []*memberlist.Node) {
	start := time.Now()
	for len(nodes) > 0 {
		remaining := make([]*memberlist.Node, 0, len(nodes))
		for _, peer := range nodes {
			s := c.Check(peer)

			if IsReady(s) {
				continue
			}

			if IsFailed(s) {
				log.Println(peer.Name, "failed", s)
				continue
			}

			remaining = append(remaining, peer)
		}

		log.Printf(
			"%3s waiting for %d node(s) to finish\n",
			time.Now().Sub(start),
			len(remaining),
		)

		nodes = remaining
		time.Sleep(time.Second)
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
