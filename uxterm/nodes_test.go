package uxterm_test

import (
	"strings"
	"testing"

	"github.com/james-lawrence/bw"
	"github.com/james-lawrence/bw/agent"
	"github.com/james-lawrence/bw/uxterm"
	"github.com/stretchr/testify/require"
)

func TestNodeName(t *testing.T) {
	long := strings.Repeat("ab", 32)
	short := uxterm.NodeName(long)
	require.Len(t, short, 36) // uuid format.
	require.Equal(t, "abababab-abab-abab-abab-abababababab", short)
	require.Equal(t, "node1", uxterm.NodeName("node1"))
}

func TestModeString(t *testing.T) {
	require.Equal(t, "leader", uxterm.ModeString(agent.PeerMode_PeerModeLeader))
	require.Equal(t, "follower", uxterm.ModeString(agent.PeerMode_PeerModeFollower))
	require.Equal(t, "peer", uxterm.ModeString(agent.PeerMode_PeerModePeer))
}

func TestRenderNodes(t *testing.T) {
	var (
		leader  = agent.NewPeer("leader")
		node    = agent.NewPeer("node1")
		silent  = agent.NewPeer("silent")
		archive = &agent.Archive{DeploymentID: []byte("0123456789abcdef")}
	)

	leader.Mode = agent.PeerMode_PeerModeLeader

	completed := func(p *agent.Peer) *agent.Message {
		return agent.DeployEvent(p, &agent.Deploy{Stage: agent.Deploy_Completed, Archive: archive})
	}

	info := &agent.InfoResponse{
		Leader:    leader,
		Completed: []*agent.Message{completed(leader), completed(node)},
	}

	rendered, err := uxterm.RenderNodes(info, []*agent.Peer{leader, node, silent})
	require.NoError(t, err)

	require.Contains(t, rendered, "Leader    : "+uxterm.PeerString(leader))
	require.Contains(t, rendered, "Latest    : None")
	require.Contains(t, rendered, "Deploying : None")

	for _, column := range []string{"archive", "mode", "name", "address", "port", "deployed"} {
		require.Contains(t, rendered, column)
	}

	id := bw.RandomID(archive.DeploymentID).String()
	row := func(name string) string {
		for _, l := range strings.Split(rendered, "\n") {
			if strings.Contains(l, name) && !strings.Contains(l, "Leader") {
				return l
			}
		}
		t.Fatalf("no row for %s in %s", name, rendered)
		return ""
	}

	require.Contains(t, row("node1"), id)
	require.Contains(t, row("leader"), id)
	require.NotContains(t, row("silent"), id)
	require.Contains(t, row("silent"), "-")
}
