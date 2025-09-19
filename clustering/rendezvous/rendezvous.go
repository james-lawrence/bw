package rendezvous

import (
	"crypto/md5"
	"encoding/binary"
	"hash"
	"math/big"
	"sort"

	"github.com/hashicorp/memberlist"
)

// Auto is just a predefined byte array that can be used as a quick way
// to compute a set of nodes when one just cares about consistency.
func Auto() []byte {
	return []byte("auto")
}

// Compute computes the HRW for each node.
func Compute(key []byte, nodes []*memberlist.Node, x func(*memberlist.Node, *big.Int)) {
	for _, node := range nodes {
		h := md5.New()
		bi := big.NewInt(0)

		h.Write([]byte(node.Name))
		h.Write(key)

		bi = bi.SetBytes(h.Sum(nil))

		x(node, bi)
	}
}

// Max - finds the node with the highest hash for the given key.
func Max(key []byte, nodes []*memberlist.Node) (max *memberlist.Node) {
	var maxValue uint64
	hasher := md5.New()

	computeFast(key, nodes, hasher, func(node *memberlist.Node, hashValue uint64) {
		if hashValue > maxValue {
			maxValue = hashValue
			max = node
		}
	})

	return max
}

// MaxN - finds the nodes with the highest hash values for the given key.
func MaxN(n int, key []byte, nodes []*memberlist.Node) []*memberlist.Node {
	if n > len(nodes) {
		n = len(nodes)
	}

	results := make([]*memberlist.Node, 0, n)
	peers := make([]nodeHash, 0, len(nodes))
	hasher := md5.New()

	computeFast(key, nodes, hasher, func(node *memberlist.Node, hashValue uint64) {
		peers = append(peers, nodeHash{peer: node, val: hashValue})
	})

	sort.Slice(peers, func(i, j int) bool { return peers[i].val > peers[j].val })

	for i := 0; i < n; i++ {
		results = append(results, peers[i].peer)
	}

	return results
}

// computeFast computes rendezvous hash using uint64 instead of big.Int for better performance
func computeFast(key []byte, nodes []*memberlist.Node, hasher hash.Hash, x func(*memberlist.Node, uint64)) {
	for _, node := range nodes {
		hasher.Reset()
		hasher.Write([]byte(node.Name))
		hasher.Write(key)
		hashBytes := hasher.Sum(nil)
		hashValue := binary.BigEndian.Uint64(hashBytes[:8])
		x(node, hashValue)
	}
}

type nodeHash struct {
	peer *memberlist.Node
	val  uint64
}
