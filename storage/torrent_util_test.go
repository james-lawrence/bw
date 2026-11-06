package storage

import (
	"bytes"
	"context"
	"crypto/rand"
	"net"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/james-lawrence/bw/internal/torrentx"
	"github.com/james-lawrence/torrent"
	"github.com/stretchr/testify/require"
)

func TestClearTorrents(t *testing.T) {
	setup := func(t *testing.T, n int) (TorrentConfig, string, []string) {
		l, err := net.Listen("tcp", "127.0.0.1:0")
		require.NoError(t, err)
		t.Cleanup(func() { l.Close() })

		dir := filepath.Join(t.TempDir(), "torrents")
		c, err := NewTorrent(
			staticCluster{},
			TorrentOptionBind(torrent.NewSocketsBind(torrentx.Socket{Listener: l, Dialer: &net.Dialer{}})),
			TorrentOptionDataDir(dir),
		)
		require.NoError(t, err)
		t.Cleanup(func() { c.client.Close() })

		// oldest first.
		names := make([]string, 0, n)
		for i := 0; i < n; i++ {
			payload := make([]byte, 4096)
			_, _ = rand.Read(payload)

			up, err := c.Uploader().NewUpload(uint64(len(payload)))
			require.NoError(t, err)
			_, err = up.Upload(bytes.NewReader(payload))
			require.NoError(t, err)
			_, _, err = up.Info()
			require.NoError(t, err)

			entries, err := os.ReadDir(dir)
			require.NoError(t, err)
			for _, e := range entries {
				if !e.Type().IsRegular() {
					continue
				}

				known := false
				for _, n := range names {
					known = known || n == e.Name()
				}

				if !known {
					names = append(names, e.Name())
					ts := time.Now().Add(time.Duration(i-n) * time.Minute)
					require.NoError(t, os.Chtimes(filepath.Join(dir, e.Name()), ts, ts))
				}
			}
		}

		return c, dir, names
	}

	t.Run("drops the oldest archives using the cached metadata", func(t *testing.T) {
		c, dir, names := setup(t, 5)
		for _, n := range names {
			_, ok := TorrentUtil{}.cachedMetadata(c, n)
			require.True(t, ok, n)
		}

		TorrentUtil{}.ClearTorrents(context.Background(), c, 3)

		for _, n := range names[:2] {
			require.NoFileExists(t, filepath.Join(dir, n))
		}
		for _, n := range names[2:] {
			require.FileExists(t, filepath.Join(dir, n))
		}
	})

	t.Run("drops archives the metadata cache doesnt know", func(t *testing.T) {
		c, dir, names := setup(t, 5)
		require.NoError(t, os.RemoveAll(filepath.Join(dir, metadataDir)))
		require.NoError(t, os.MkdirAll(filepath.Join(dir, metadataDir), 0700))
		_, ok := TorrentUtil{}.cachedMetadata(c, names[0])
		require.False(t, ok)

		TorrentUtil{}.ClearTorrents(context.Background(), c, 3)

		for _, n := range names[:2] {
			require.NoFileExists(t, filepath.Join(dir, n))
		}
		for _, n := range names[2:] {
			require.FileExists(t, filepath.Join(dir, n))
		}
	})
}
