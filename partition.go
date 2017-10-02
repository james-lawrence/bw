package bw

import (
	"log"
	"math"

	yaml "gopkg.in/yaml.v1"
)

const (
	// PartitionStrategyPercentage percentage based partition strategy.
	PartitionStrategyPercentage = "percent"
	// PartitionStrategyBatch batch based partition strategy.
	PartitionStrategyBatch = "batch"
)

// PartitionerFromConfig ...
func PartitionerFromConfig(strategy string, serialized []byte) Partitioner {
	def := ConstantPartitioner{BatchMax: 1}
	switch strategy {
	case PartitionStrategyPercentage:
		var b PercentPartitioner
		return newPartitionFromConfig(serialized, &b, def)
	case PartitionStrategyBatch:
		var b ConstantPartitioner
		return newPartitionFromConfig(serialized, &b, def)
	default:
		log.Printf("unknown strategy: %s, defaulting to one at a time\n", strategy)
		return def
	}
}

func newPartitionFromConfig(serialized []byte, v, def Partitioner) (_ Partitioner) {
	if err := yaml.Unmarshal(serialized, v); err != nil {
		log.Println("failed to parse partition strategy, falling back to default", err)
		return def
	}

	return v
}

// Partitioner determines the number of nodes to simultaneously deploy to
// based on the total number of nodes.
type Partitioner interface {
	Partition(length int) (size int)
}

// PercentPartitioner size is based on the percentage. has an upper bound of 1.0.
type PercentPartitioner struct {
	Percentage float64
}

// Partition ...
func (t PercentPartitioner) Partition(length int) int {
	ratio := math.Min(float64(t.Percentage), 1.0)
	computed := int(math.Max(math.Floor(float64(length)*ratio), 1.0))
	return computed
}

// ConstantPartitioner partition will return the specified min(length, size).
type ConstantPartitioner struct {
	BatchMax int
}

// Partition implements partitioner
func (t ConstantPartitioner) Partition(length int) int {
	return max(1, min(length, int(t.BatchMax)))
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
