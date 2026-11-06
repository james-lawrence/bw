package storage

import (
	"context"
	"encoding/hex"
	"log"
	"os"
	"path/filepath"
	"strings"

	"github.com/james-lawrence/bw/internal/debugx"
	"github.com/james-lawrence/bw/internal/errorsx"
	"github.com/james-lawrence/bw/internal/fsx"
	"github.com/james-lawrence/bw/internal/iterx"
	"github.com/james-lawrence/torrent"
	"github.com/james-lawrence/torrent/dht/int160"
	"github.com/james-lawrence/torrent/metainfo"
	"github.com/pkg/errors"
)

// TorrentUtil utility functions for the torrent storage subsystem.
type TorrentUtil struct{}

// ClearTorrents periodically flushes torrents from storage based on whether or not
// the deployment directory is still around.
func (t TorrentUtil) ClearTorrents(ctx context.Context, c TorrentConfig, keepN int) {
	debugx.Println("torrent data directory", c.dir)
	dropped := 0

	entries, err := os.ReadDir(c.dir)
	if err != nil {
		log.Println("unable to read torrent data directory", err)
		return
	}

	paths := make([]string, 0, len(entries))
	for _, e := range entries {
		if e.Type().IsRegular() {
			paths = append(paths, filepath.Join(c.dir, e.Name()))
		}
	}

	stale := fsx.KeepNewestN(keepN, iterx.Slice(paths...))
	for path := range stale.Each(ctx) {
		// loading a torrent from its file hashes the entire archive, the metadata cache already has what's needed.
		md, ok := t.cachedMetadata(c, filepath.Base(path))
		if !ok {
			var cause error
			if md, cause = torrent.NewFromFile(path); cause != nil {
				log.Println("unable to load torrent", path, cause)
				continue
			}
		}

		errorsx.Log(c.client.Stop(md))
		errorsx.Log(os.Remove(path))
		dropped = dropped + 1
	}
	errorsx.Log(stale.Err())

	log.Println(dropped, "torrents dropped, keeping the", keepN, "newest")
}

// cachedMetadata finds the metadata of the torrent stored in the named file. the cache is a directory of small
// <infohash>.torrent files, the infohash can't be derived from the archive so match on the name of the torrent.
// the store's Each doesn't walk those files so list the directory.
func (TorrentUtil) cachedMetadata(c TorrentConfig, name string) (md torrent.Metadata, found bool) {
	if c.metadata == nil {
		return md, false
	}

	entries, err := os.ReadDir(filepath.Join(c.dir, metadataDir))
	if err != nil {
		return md, false
	}

	for _, e := range entries {
		decoded, err := hex.DecodeString(strings.TrimSuffix(e.Name(), ".torrent"))
		if err != nil || e.IsDir() {
			continue
		}

		if md, err = c.metadata.Read(int160.FromBytes(decoded)); err == nil && md.DisplayName == name {
			return md, true
		}
	}

	return torrent.Metadata{}, false
}

// PrintTorrentInfo prints information about the torrent cluster.
func (t TorrentUtil) PrintTorrentInfo(c TorrentConfig) {
	t.printTorrentInfo(c.client)
}

func (TorrentUtil) printTorrentInfo(c *torrent.Client) {
	log.Println(c.String())
}

func (t TorrentUtil) loadDir(dir string, c *torrent.Client) error {
	return filepath.Walk(dir, func(path string, info os.FileInfo, err error) error {
		const (
			piecesDB = ".torrent.bolt.db"
		)

		if err != nil {
			return err
		}

		// nothing to do on the root directory.
		if path == dir {
			return nil
		}

		// only care about the torrents in the current directory.
		if info.IsDir() {
			return filepath.SkipDir
		}

		// ignore the database file
		if info.Name() == piecesDB {
			return nil
		}

		if _, err = t.loadTorrent(c, path); err != nil {
			return err
		}

		return nil
	})
}

// func (TorrentUtil) debugDHT() dht.ServerConfig {
// 	return dht.ServerConfig{
// 		OnQuery: func(query *krpc.Msg, source net.Addr) bool {
// 			log.Println("query", source.String(), spew.Sdump(query))
// 			return true
// 		},
// 		OnAnnouncePeer: func(infoHash metainfo.Hash, ip net.IP, port int, portOK bool) {
// 			log.Printf("announce peer %s:%d %t %s\n", ip.String(), port, portOK, infoHash.String())
// 		},
// 	}
// }

// func (TorrentUtil) filePath(dir, name string) string {
// 	return filepath.Join(dir, name)
// }

// func (t TorrentUtil) createTorrent(dir, name string) (*os.File, error) {
// 	return t.createFile(t.filePath(dir, name))
// }

func (TorrentUtil) createFile(dir string) (*os.File, error) {
	var (
		err error
		dst *os.File
	)

	if err = os.MkdirAll(filepath.Join(dir), 0755); err != nil {
		return nil, errors.WithStack(err)
	}

	if dst, err = os.CreateTemp(dir, "upload-*.bin"); err != nil {
		return nil, errors.WithStack(err)
	}

	return dst, err
}

func (t TorrentUtil) loadTorrent(c *torrent.Client, path string) (m torrent.Metadata, err error) {
	var (
		tt torrent.Torrent
	)

	if tt, _, err = c.MaybeStart(torrent.NewFromFile(path)); err != nil {
		return m, err
	}

	return tt.Metadata(), nil
}

func (TorrentUtil) magnet(meta torrent.Metadata) metainfo.Magnet {
	return torrent.NewMagnet(meta)
}
