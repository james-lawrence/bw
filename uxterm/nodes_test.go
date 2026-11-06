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

func TestModeString(t *testing.T) {
	require.Equal(t, "leader", uxterm.ModeString(agent.PeerMode_PeerModeLeader))
	require.Equal(t, "follower", uxterm.ModeString(agent.PeerMode_PeerModeFollower))
	require.Equal(t, "peer", uxterm.ModeString(agent.PeerMode_PeerModePeer))
}
