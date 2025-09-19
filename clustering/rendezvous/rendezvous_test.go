package rendezvous_test

import (
	"net"

	"github.com/hashicorp/memberlist"
	"github.com/james-lawrence/bw/clustering/clusteringtestutil"
	. "github.com/james-lawrence/bw/clustering/rendezvous"

	. "github.com/onsi/ginkgo/v2"

	. "github.com/onsi/gomega"
)

var _ = Describe("Rendezvous", func() {
	const (
		sampleKey1 = "hello world"
	)

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

		It("should return in the same order as Max", func() {
			nodes := []*memberlist.Node{
				clusteringtestutil.NewNode("node-3", net.ParseIP("127.0.0.3")),
				clusteringtestutil.NewNode("node-4", net.ParseIP("127.0.0.4")),
				clusteringtestutil.NewNode("node-2", net.ParseIP("127.0.0.2")),
				clusteringtestutil.NewNode("node-5", net.ParseIP("127.0.0.5")),
				clusteringtestutil.NewNode("node-1", net.ParseIP("127.0.0.1")),
			}

			Expect(Max([]byte(sampleKey1), nodes)).To(Equal(MaxN(1, []byte(sampleKey1), nodes)[0]))
		})
	})
})
