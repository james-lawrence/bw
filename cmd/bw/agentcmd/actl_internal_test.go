package agentcmd

import (
	"net"
	"testing"

	"github.com/james-lawrence/bw/agent"
	"github.com/james-lawrence/bw/clustering"
	"github.com/stretchr/testify/require"
)

func TestControlConnectionFilters(t *testing.T) {
	peers := []*agent.Peer{
		{Name: "alpha", Ip: "10.0.0.1"},
		{Name: "beta", Ip: "10.0.0.2"},
		{Name: "gamma", Ip: "10.0.0.3"},
	}

	match := func(t *testing.T, c controlConnection) (r []string) {
		for _, p := range peers {
			for _, f := range c.filters() {
				if f.Match(p) {
					r = append(r, p.Name)
					break
				}
			}
		}
		return r
	}

	t.Run("no filters matches everything", func(t *testing.T) {
		require.Equal(t, []string{"alpha", "beta", "gamma"}, match(t, controlConnection{}))
	})

	t.Run("ip", func(t *testing.T) {
		require.Equal(t, []string{"beta"}, match(t, controlConnection{IPs: []net.IP{net.ParseIP("10.0.0.2")}}))
	})
}

func TestControlConnectionFiltered(t *testing.T) {
	peers := []*agent.Peer{
		{Name: "alpha", Ip: "10.0.0.1"},
		{Name: "beta", Ip: "10.0.0.2"},
	}
	c := clustering.NewStatic(agent.PeersToNodes(peers...)...)

	t.Run("no filters keeps every member", func(t *testing.T) {
		require.Len(t, controlConnection{}.filtered(c).Members(), 2)
	})

	t.Run("ip filter narrows members", func(t *testing.T) {
		m := controlConnection{IPs: []net.IP{net.ParseIP("10.0.0.2")}}.filtered(c).Members()
		require.Len(t, m, 1)
		require.Equal(t, "beta", m[0].Name)
	})
}
