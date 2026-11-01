package storage

import (
	"bytes"
	"context"
	"io"
	"net"
	"path/filepath"
	"testing"
	"time"

	"github.com/james-lawrence/bw/agent"
	"github.com/james-lawrence/bw/internal/torrentx"
	"github.com/james-lawrence/torrent"
	"github.com/stretchr/testify/require"
)

type staticCluster struct {
	peers []*agent.Peer
	local *agent.Peer
}

func (t staticCluster) Quorum() []*agent.Peer { return t.peers }
func (t staticCluster) Local() *agent.Peer    { return t.local }

func newLoopbackTorrent(t *testing.T, cls func(port uint32) staticCluster) (TorrentConfig, *agent.Peer) {
	l, err := net.Listen("tcp", "127.0.0.1:0")
	require.NoError(t, err)
	t.Cleanup(func() { l.Close() })

	me := &agent.Peer{Ip: "127.0.0.1", P2PPort: uint32(l.Addr().(*net.TCPAddr).Port)}
	c, err := NewTorrent(
		cls(me.P2PPort),
		TorrentOptionBind(torrent.NewSocketsBind(torrentx.Socket{Listener: l, Dialer: &net.Dialer{}})),
		TorrentOptionDataDir(filepath.Join(t.TempDir(), "torrents")),
	)
	require.NoError(t, err)
	t.Cleanup(func() { c.client.Close() })
	return c, me
}

// seeder uploads an archive, the downloader fetches it by magnet from the seeder.
func TestTorrentLoopbackTransfer(t *testing.T) {
	var seederPeer *agent.Peer
	seeder, seederPeer := newLoopbackTorrent(t, func(port uint32) staticCluster {
		return staticCluster{}
	})

	payload := bytes.Repeat([]byte("loopback torrent payload "), 100000)
	up, err := seeder.Uploader().NewUpload(uint64(len(payload)))
	require.NoError(t, err)
	_, err = up.Upload(bytes.NewReader(payload))
	require.NoError(t, err)
	_, magnet, err := up.Info()
	require.NoError(t, err)

	downloader, _ := newLoopbackTorrent(t, func(uint32) staticCluster {
		return staticCluster{peers: []*agent.Peer{seederPeer}}
	})

	ctx, cancel := context.WithTimeout(context.Background(), 20*time.Second)
	defer cancel()

	r := downloader.Downloader().New().Download(ctx, &agent.Archive{Location: magnet})
	defer r.Close()

	got, err := io.ReadAll(r)
	require.NoError(t, err)
	require.True(t, bytes.Equal(payload, got), "downloaded %d bytes, expected %d", len(got), len(payload))
}
