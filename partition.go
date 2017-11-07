package bw

import (
	"math"
)

const (
	// PartitionStrategyPercentage percentage based partition strategy.
	PartitionStrategyPercentage = "percent"
	// PartitionStrategyBatch batch based partition strategy.
	PartitionStrategyBatch = "batch"
)

// PartitionFromFloat64 generates a partitioner from a float value.
// rules:
// default, one at a time: x == 0
// deploy x percent: 0 < x < 1.0
// deploy batch of floor(x): 1.0 < x < inf
func PartitionFromFloat64(p float64) Partitioner {
	switch {
	case p > 0 && p <= 1.0:
		return PercentPartitioner(p)
	case p >= 1.0:
		x := int(math.Floor(p))
		return ConstantPartitioner(x)
	default:
		return ConstantPartitioner(1)
	}
}

// Partitioner determines the number of nodes to simultaneously deploy to
// based on the total number of nodes.
type Partitioner interface {
	Partition(length int) (size int)
}

// PercentPartitioner size is based on the percentage. has an upper bound of 1.0.
type PercentPartitioner float64

// Partition ...
func (t PercentPartitioner) Partition(length int) int {
	ratio := math.Min(float64(t), 1.0)
	computed := int(math.Max(math.Floor(float64(length)*ratio), 1.0))
	return computed
}

// ConstantPartitioner partition will return the specified min(length, size).
type ConstantPartitioner int

// Partition implements partitioner
func (t ConstantPartitioner) Partition(length int) int {
	return max(1, min(length, int(t)))
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
