package rendezvous

import (
	"crypto/md5"
	"math/big"
	"slices"

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
	maxValue := big.NewInt(0)

	Compute(key, nodes, func(node *memberlist.Node, bi *big.Int) {
		if bi.Cmp(maxValue) == 1 {
			maxValue = bi
			max = node
		}
	})

	return max
}

// MaxN - finds the node with the highest hash for the given key.
func MaxN(n int, key []byte, nodes []*memberlist.Node) []*memberlist.Node {
	type pair struct {
		peer *memberlist.Node
		val  *big.Int
	}

	if n > len(nodes) {
		n = len(nodes)
	}

	results := make([]*memberlist.Node, 0, n)
	peers := make([]pair, 0, len(nodes))

	Compute(key, nodes, func(node *memberlist.Node, bi *big.Int) {
		peers = append(peers, pair{peer: node, val: bi})
	})

	slices.SortStableFunc(peers, func(a, b pair) int { return b.val.Cmp(a.val) })

	for _, p := range peers[:n] {
		results = append(results, p.peer)
	}

	return results
}
