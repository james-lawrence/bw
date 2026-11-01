package storage

import (
	"testing"

	"github.com/james-lawrence/bw/agent"
	"github.com/james-lawrence/torrent/dht/int160"
	"github.com/stretchr/testify/require"
)

func TestPeerToNode(t *testing.T) {
	a, ok := peerToNode(&agent.Peer{Ip: "127.0.0.1", P2PPort: 2001})
	require.True(t, ok)
	b, ok := peerToNode(&agent.Peer{Ip: "127.0.0.3", P2PPort: 2001})
	require.True(t, ok)

	// the zero id is treated as ourselves by the client, which prevents dialing.
	require.NotEqual(t, int160.Zero(), a.ID)
	require.NotEqual(t, a.ID, b.ID)

	_, ok = peerToNode(&agent.Peer{Ip: "garbage"})
	require.False(t, ok)
}
