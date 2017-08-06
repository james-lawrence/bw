package rendezvous_test

import (
	"net"

	"bitbucket.org/jatone/bearded-wookie/clustering/clusteringtestutil"

	. "bitbucket.org/jatone/bearded-wookie/clustering/rendezvous"
	"github.com/hashicorp/memberlist"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/ginkgo/extensions/table"
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
			clusteringtestutil.NewPeer("node-1", net.ParseIP("127.0.0.1")),
			clusteringtestutil.NewPeers(1),
		),
		Entry(
			"multiple nodes", sampleKey1,
			clusteringtestutil.NewPeer("node-1", net.ParseIP("127.0.0.1")),
			clusteringtestutil.NewPeers(5),
		),
		Entry(
			"different keys should return different nodes", "hello world - 1",
			clusteringtestutil.NewPeer("node-5", net.ParseIP("127.0.0.5")),
			clusteringtestutil.NewPeers(5),
		),
	)

	Describe("MaxN", func() {
		const peers = 5
		DescribeTable("cluster",
			func(n int, key string, expectedPeers ...*memberlist.Node) {
				Expect(MaxN(n, []byte(key), clusteringtestutil.NewPeers(peers))).To(Equal(expectedPeers))
			},
			Entry(
				"should return every node if n is larger than the number of peers", 2*peers, sampleKey1,
				clusteringtestutil.NewPeer("node-3", net.ParseIP("127.0.0.3")),
				clusteringtestutil.NewPeer("node-4", net.ParseIP("127.0.0.4")),
				clusteringtestutil.NewPeer("node-2", net.ParseIP("127.0.0.2")),
				clusteringtestutil.NewPeer("node-5", net.ParseIP("127.0.0.5")),
				clusteringtestutil.NewPeer("node-1", net.ParseIP("127.0.0.1")),
			),
			Entry(
				"example 1", 1, sampleKey1,
				clusteringtestutil.NewPeer("node-3", net.ParseIP("127.0.0.3")),
			),
			Entry(
				"example 2", 3, sampleKey1,
				clusteringtestutil.NewPeer("node-3", net.ParseIP("127.0.0.3")),
				clusteringtestutil.NewPeer("node-4", net.ParseIP("127.0.0.4")),
				clusteringtestutil.NewPeer("node-2", net.ParseIP("127.0.0.2")),
			),
		)
	})
})
