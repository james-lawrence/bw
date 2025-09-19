package rendezvous_test

import (
	"net"
	"testing"

	"github.com/hashicorp/memberlist"
	"github.com/james-lawrence/bw/clustering/clusteringtestutil"
	. "github.com/james-lawrence/bw/clustering/rendezvous"

	. "github.com/onsi/ginkgo/v2"

	. "github.com/onsi/gomega"
)

var _ = Describe("Rendezvous", func() {
	const sampleKey1 = "hello world"
	DescribeTable("Max",
		func(key string, expectedPeer *memberlist.Node, peers []*memberlist.Node) {
			Expect(Max([]byte(key), peers)).To(Equal(expectedPeer))
		},
		Entry(
			"single node", sampleKey1,
			clusteringtestutil.NewNode("node-1", net.ParseIP("127.0.0.1")),
			clusteringtestutil.NewNodes(1),
		),
		Entry(
			"multiple nodes", sampleKey1,
			clusteringtestutil.NewNode("node-1", net.ParseIP("127.0.0.1")),
			clusteringtestutil.NewNodes(5),
		),
		Entry(
			"different keys should return different nodes", "hello world - 1",
			clusteringtestutil.NewNode("node-5", net.ParseIP("127.0.0.5")),
			clusteringtestutil.NewNodes(5),
		),
	)

	Describe("MaxN", func() {
		const peers = 5
		DescribeTable("cluster",
			func(n int, key string, expectedPeers ...*memberlist.Node) {
				Expect(MaxN(n, []byte(key), clusteringtestutil.NewNodes(peers))).To(Equal(expectedPeers))
			},
			Entry(
				"should return every node if n is larger than the number of peers", 2*peers, sampleKey1,
				clusteringtestutil.NewNode("node-1", net.ParseIP("127.0.0.1")),
				clusteringtestutil.NewNode("node-5", net.ParseIP("127.0.0.5")),
				clusteringtestutil.NewNode("node-2", net.ParseIP("127.0.0.2")),
				clusteringtestutil.NewNode("node-4", net.ParseIP("127.0.0.4")),
				clusteringtestutil.NewNode("node-3", net.ParseIP("127.0.0.3")),
			),
			Entry(
				"example 1", 1, sampleKey1,
				clusteringtestutil.NewNode("node-1", net.ParseIP("127.0.0.1")),
			),
			Entry(
				"example 2", 3, sampleKey1,
				clusteringtestutil.NewNode("node-1", net.ParseIP("127.0.0.1")),
				clusteringtestutil.NewNode("node-5", net.ParseIP("127.0.0.5")),
				clusteringtestutil.NewNode("node-2", net.ParseIP("127.0.0.2")),
			),
		)
	})
})

// createTestNodes creates a slice of test nodes for benchmarking
func createTestNodes(count int) []*memberlist.Node {
	nodes := make([]*memberlist.Node, count)
	for i := range count {
		nodes[i] = &memberlist.Node{
			Name: string(rune('a'+i%26)) + string(rune('a'+(i/26)%26)),
		}
	}
	return nodes
}

func BenchmarkMax(b *testing.B) {
	nodes := createTestNodes(100)
	key := []byte("test-key")

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = Max(key, nodes)
	}
}

func BenchmarkMaxN(b *testing.B) {
	nodes := createTestNodes(100)
	key := []byte("test-key")
	n := 10

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = MaxN(n, key, nodes)
	}
}
