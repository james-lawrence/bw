package storage

import (
	"bytes"
	"context"
	"crypto/rand"
	"fmt"
	"net"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/james-lawrence/bw/internal/testingx"
	"github.com/james-lawrence/bw/internal/torrentx"
	"github.com/james-lawrence/torrent"
)

// BenchmarkClearTorrents runs after every deploy, deployments are single file torrents stored in the data directory,
// the file is the deployment archive. ClearTorrents builds the torrent of every stale archive from the file
// (torrent.NewFromFile) before removing it, which reads the whole archive. reports the bytes read per call.
func BenchmarkClearTorrents(b *testing.B) {
	const (
		keep  = 3
		stale = 2
	)

	for _, size := range []int{1 << 20, 32 << 20} {
		b.Run(fmt.Sprintf("archive-%dMiB", size>>20), func(b *testing.B) {
			l, err := net.Listen("tcp", "127.0.0.1:0")
			if err != nil {
				b.Fatal(err)
			}
			defer l.Close()

			dir := filepath.Join(testingx.TempDir(b), "torrents")
			c, err := NewTorrent(
				staticCluster{},
				TorrentOptionBind(torrent.NewSocketsBind(torrentx.Socket{Listener: l, Dialer: &net.Dialer{}})),
				TorrentOptionDataDir(dir),
			)
			if err != nil {
				b.Fatal(err)
			}
			defer c.client.Close()

			var (
				util   TorrentUtil
				reads  uint64
				writes uint64
			)

			// keep+stale archives, oldest first, uploaded the way deploys are so the client knows them.
			populate := func() {
				for i := 0; i < keep+stale; i++ {
					payload := make([]byte, size)
					_, _ = rand.Read(payload)

					up, err := c.Uploader().NewUpload(uint64(size))
					if err != nil {
						b.Fatal(err)
					}
					if _, err = up.Upload(bytes.NewReader(payload)); err != nil {
						b.Fatal(err)
					}
					if _, _, err = up.Info(); err != nil {
						b.Fatal(err)
					}
				}

				// order the archives by age, the upload order.
				entries := mustReadDir(b, dir)
				n := 0
				for _, e := range entries {
					if !e.Type().IsRegular() {
						continue
					}
					ts := time.Now().Add(time.Duration(n-keep-stale) * time.Minute)
					if err := os.Chtimes(filepath.Join(dir, e.Name()), ts, ts); err != nil {
						b.Fatal(err)
					}
					n++
				}
			}

			b.ResetTimer()
			for i := 0; i < b.N; i++ {
				b.StopTimer()
				populate()
				before := testingx.ReadIOCounters()
				b.StartTimer()

				util.ClearTorrents(context.Background(), c, keep)

				b.StopTimer()
				d := testingx.ReadIOCounters().Sub(before)
				reads, writes = reads+d.ReadChars, writes+d.WriteChars
				for _, e := range mustReadDir(b, dir) {
					if e.Type().IsRegular() {
						_ = os.Remove(filepath.Join(dir, e.Name()))
					}
				}
				b.StartTimer()
			}
			b.StopTimer()

			b.ReportMetric(float64(reads)/float64(b.N), "rchar-B/op")
			b.ReportMetric(float64(writes)/float64(b.N), "wchar-B/op")
			b.ReportMetric(float64(reads)/float64(b.N)/float64(size*stale), "archive-reads/stale-archive")
		})
	}
}

func mustReadDir(b *testing.B, dir string) []os.DirEntry {
	b.Helper()

	entries, err := os.ReadDir(dir)
	if err != nil {
		b.Fatal(err)
	}
	return entries
}
