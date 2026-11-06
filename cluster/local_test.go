package cluster_test

import (
	"net"
	"testing"

	"github.com/james-lawrence/bw/agent"
	"github.com/james-lawrence/bw/cluster"
	"github.com/stretchr/testify/require"
	"gopkg.in/yaml.v3"
)

// nodes advertise whether they archive deployments through their gossiped metadata.
func TestLocalAdvertisesArchive(t *testing.T) {
	cases := []struct {
		name     string
		config   string
		expected bool
	}{
		{name: "archive directory configured", config: "bootstrap:\n  archiveDirectory: /tmp/archive\n", expected: true},
		{name: "no archive directory", config: "bootstrap:\n  readonly: false\n", expected: false},
		{name: "readonly archive directory", config: "bootstrap:\n  archiveDirectory: /tmp/archive\n  readonly: true\n", expected: false},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			c := agent.NewConfig(agent.ConfigOptionAdvertised(&net.TCPAddr{IP: net.IPv4(127, 0, 0, 1), Port: 2000}))
			require.NoError(t, yaml.Unmarshal([]byte(tc.config), &c))

			p := c.Peer()
			p.Name = "node1"
			n := agent.PeerToNode(p)
			n.Meta = cluster.NewEventsQueue(cluster.NewLocal(p)).NodeMeta(512)

			peer, err := agent.NodeToPeer(n)
			require.NoError(t, err)
			require.Equal(t, tc.expected, peer.Archive)
		})
	}
}
