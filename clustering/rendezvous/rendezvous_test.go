package rendezvous_test

import (
	"io"
	"log"
	"net"
	"os"
	"testing"

	"github.com/hashicorp/memberlist"
	"github.com/james-lawrence/bw/clustering/clusteringtestutil"
	. "github.com/james-lawrence/bw/clustering/rendezvous"
	"github.com/stretchr/testify/require"
)

const sampleKey1 = "hello world"

func TestMain(m *testing.M) {
	log.SetOutput(io.Discard)
	os.Exit(m.Run())
}

func TestMax(t *testing.T) {
	for _, c := range []struct {
		name     string
		key      string
		expected *memberlist.Node
		peers    []*memberlist.Node
	}{
		{
			"single node", sampleKey1,
			clusteringtestutil.NewNode("node-1", net.ParseIP("127.0.0.1")),
			clusteringtestutil.NewNodes(1),
		},
		{
			"multiple nodes", sampleKey1,
			clusteringtestutil.NewNode("node-1", net.ParseIP("127.0.0.1")),
			clusteringtestutil.NewNodes(5),
		},
		{
			"different keys should return different nodes", "hello world - 1",
			clusteringtestutil.NewNode("node-5", net.ParseIP("127.0.0.5")),
			clusteringtestutil.NewNodes(5),
		},
	} {
		t.Run(c.name, func(t *testing.T) {
			require.Equal(t, c.expected, Max([]byte(c.key), c.peers))
		})
	}
}

func TestMaxN(t *testing.T) {
	const peers = 5

	t.Run("cluster", func(t *testing.T) {
		for _, c := range []struct {
			name     string
			n        int
			key      string
			expected []*memberlist.Node
		}{
			{
				"should return every node if n is larger than the number of peers", 2 * peers, sampleKey1,
				[]*memberlist.Node{
					clusteringtestutil.NewNode("node-1", net.ParseIP("127.0.0.1")),
					clusteringtestutil.NewNode("node-5", net.ParseIP("127.0.0.5")),
					clusteringtestutil.NewNode("node-2", net.ParseIP("127.0.0.2")),
					clusteringtestutil.NewNode("node-4", net.ParseIP("127.0.0.4")),
					clusteringtestutil.NewNode("node-3", net.ParseIP("127.0.0.3")),
				},
			},
			{
				"example 1", 1, sampleKey1,
				[]*memberlist.Node{
					clusteringtestutil.NewNode("node-1", net.ParseIP("127.0.0.1")),
				},
			},
			{
				"example 2", 3, sampleKey1,
				[]*memberlist.Node{
					clusteringtestutil.NewNode("node-1", net.ParseIP("127.0.0.1")),
					clusteringtestutil.NewNode("node-5", net.ParseIP("127.0.0.5")),
					clusteringtestutil.NewNode("node-2", net.ParseIP("127.0.0.2")),
				},
			},
		} {
			t.Run(c.name, func(t *testing.T) {
				require.Equal(t, c.expected, MaxN(c.n, []byte(c.key), clusteringtestutil.NewNodes(peers)))
			})
		}
	})

	t.Run("should return in the same order as Max", func(t *testing.T) {
		nodes := []*memberlist.Node{
			clusteringtestutil.NewNode("node-3", net.ParseIP("127.0.0.3")),
			clusteringtestutil.NewNode("node-4", net.ParseIP("127.0.0.4")),
			clusteringtestutil.NewNode("node-2", net.ParseIP("127.0.0.2")),
			clusteringtestutil.NewNode("node-5", net.ParseIP("127.0.0.5")),
			clusteringtestutil.NewNode("node-1", net.ParseIP("127.0.0.1")),
		}

		require.Equal(t, MaxN(1, []byte(sampleKey1), nodes)[0], Max([]byte(sampleKey1), nodes))
	})
}
