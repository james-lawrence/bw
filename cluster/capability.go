package cluster

import "github.com/james-lawrence/bw/x/debugx"

const (
	// Passive represents a node that communicates with the cluster but isn't considered an actual
	// member. Useful for agents that perform things like monitoring, commandline interfaces, etc.
	// by default a node is not passive.
	Passive int = iota

	// Node represents an active member of the cluster
	Node

	// LastCapability just marks the maximum ability integer value.
	// useful for looping over abilities: for i := 0; i < LastCapability; i++
	LastCapability
)

// ZeroBitField represents an empty bitfield.
var zeroBitField = []byte{}

// Capability interface for determining the capability of the cluster.
type Capability interface {
	Has(ability int) bool
}

// NewCapability - creates a Capability from a bitfield vector.
func NewCapability(abilities ...int) Capability {
	return bitField(NewBitField(abilities...))
}

// NewBitField - creates a bitfield from a set of abilities.
func NewBitField(abilities ...int) []byte {
	return BitFieldMerge(zeroBitField, abilities...)
}

// BitField creates a Capability from the bitfield vector.
func BitField(vector []byte, abilities ...int) Capability {
	return bitField(BitFieldMerge(vector, abilities...))
}

// BitField implements the capability interface over an array of bytes
// representing a bitfield.
type bitField []byte

// Has - see Capability
func (t bitField) Has(ability int) bool {
	quo, rem := offsets(ability)

	// if ability is out of range
	if quo >= len(t) {
		return false
	}

	bitflags := t[quo]
	actual := byte(1 << uint(1*rem))

	debugx.Printf("bitflags[%.8b], actual[%.8b]\n", bitflags, actual)
	// 11111111&00010000 = 00010000
	return bitflags&actual == actual
}

func offsets(i int) (quo, rem int) {
	return int(i / 8), int(i % 8)
}

// BitFieldMerge merges a bitfield with a set of abilities.
func BitFieldMerge(vector []byte, abilities ...int) []byte {
	max := len(vector)

	// determine the maximum size of the vector by finding the max quotient.
	for _, ability := range abilities {
		if quo, _ := offsets(ability); quo > max {
			max = quo
		}
	}

	if max == 0 && len(abilities) > 0 {
		max = 1
	}

	if len(vector) < max {
		// create the vector using the max.
		buf := make([]byte, max)
		copy(buf, vector)
		vector = buf
	}

	for _, ability := range abilities {
		quo, rem := offsets(ability)
		vector[quo] = vector[quo] | byte(1<<uint(1*rem))
	}

	return vector
}
