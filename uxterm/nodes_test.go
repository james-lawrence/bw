package uxterm_test

import (
	"strings"
	"testing"

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

func TestNodeMode(t *testing.T) {
	leader, follower, other := agent.NewPeer("leader"), agent.NewPeer("follower"), agent.NewPeer("other")
	info := &agent.InfoResponse{Leader: leader, Quorum: []*agent.Peer{leader, follower}}

	require.Equal(t, "leader", uxterm.NodeMode(info, leader))
	require.Equal(t, "follower", uxterm.NodeMode(info, follower))
	require.Equal(t, "peer", uxterm.NodeMode(info, other))
	require.Equal(t, "quorum", uxterm.NodeMode(&agent.InfoResponse{Quorum: []*agent.Peer{follower}}, follower))
}
