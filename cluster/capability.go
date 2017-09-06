package cluster

import "log"

const (
	// Deploy represents a node that communicates with the cluster but isn't an actual
	// member. Useful for agents that perform things like monitoring, commandline interfaces, etc.
	// by default a node is not a lurker.
	Deploy int = iota
	// LastCapability just marks the maximum ability integer value.
	// useful for looping over abilities: for i := 0; i < LastCapability; i++
	LastCapability
)

// Capability interface for determining the capability of the cluster.
type Capability interface {
	Has(ability int) bool
}

// NewBitField - creates a Capability from a bitfield vector.
func NewBitField(abilities ...int) Capability {
	return bitField(BitFieldMerge([]byte{}, abilities...))
}

// BitField creates a Capability from the bitfield vector.
func BitField(vector []byte, abilities ...int) Capability {
	if len(abilities) == 0 {
		return bitField(vector)
	}
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

	log.Printf("bitflags[%.8b], actual[%.8b]\n", bitflags, actual)
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
