package agentcmd_test

import (
	"context"
	"net"
	"os"
	"path/filepath"
	"sync"
	"testing"

	"github.com/james-lawrence/bw"
	"github.com/james-lawrence/bw/agenttestutil"
	"github.com/james-lawrence/bw/internal/rsax"
	"github.com/stretchr/testify/require"
)

// the torrent subsystem must keep everything it reads and writes within the torrent directory of the agent,
// not leak into the user's cache directory.
func TestDaemonTorrentStaysInTorrentDirectory(t *testing.T) {
	if testing.Short() {
		t.Skip("boots an agent, skipping in short mode")
	}

	cache := t.TempDir()
	t.Setenv("CACHE_DIRECTORY", cache)
	t.Setenv("XDG_CACHE_HOME", cache)

	l, err := net.Listen("tcp", "127.0.0.1:0")
	require.NoError(t, err)
	addr := l.Addr().(*net.TCPAddr)
	require.NoError(t, l.Close())

	// not t.TempDir(), unix socket paths have a limited length.
	dir, err := os.MkdirTemp("", "bw.test.agent.*")
	require.NoError(t, err)
	t.Cleanup(func() { _ = os.RemoveAll(dir) })

	seed := rsax.DeterministcSeed("bw.test", "daemon", t.Name())
	client := sync.OnceValues(func() ([]byte, error) { return rsax.Deterministic(seed, rsax.UnsafeBits()) })

	require.NoError(t, agenttestutil.AgentIn(t, dir, client, addr))
	require.NoError(t, agenttestutil.Client(t, client, addr))
	require.NoError(t, agenttestutil.WaitQuorum(context.Background()))

	entries, err := os.ReadDir(cache)
	require.NoError(t, err)
	require.Empty(t, entries, "torrent leaked into the cache directory")

	torrents := filepath.Join(dir, "a", bw.DirTorrents)
	require.DirExists(t, filepath.Join(torrents, ".bitmaps"))
	require.DirExists(t, filepath.Join(torrents, ".metadata"))
}
