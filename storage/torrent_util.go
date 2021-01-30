package storage

import (
	"io/ioutil"
	"log"
	"net"
	"os"
	"path/filepath"

	"github.com/davecgh/go-spew/spew"
	"github.com/james-lawrence/bw"
	"github.com/james-lawrence/bw/internal/x/debugx"
	"github.com/james-lawrence/bw/internal/x/logx"
	"github.com/james-lawrence/torrent"
	"github.com/james-lawrence/torrent/dht/v2"
	"github.com/james-lawrence/torrent/dht/v2/krpc"
	"github.com/james-lawrence/torrent/metainfo"
	"github.com/pkg/errors"
)

// TorrentUtil utility functions for the torrent storage subsystem.
type TorrentUtil struct{}

// Bootstrap the torrent client
func (TorrentUtil) Bootstrap(c *torrent.Client) {
	var (
		err   error
		stats dht.TraversalStats
	)

	for _, s := range c.DhtServers() {
		if stats, err = s.Bootstrap(); err != nil {
			log.Println("failed to bootstrap dht server", err)
			continue
		}

		log.Println("dht bootstrap stats", spew.Sdump(stats))
	}
}

// ClearTorrents periodically flushes torrents from storage based on whether or not
// the deployment directory is still around.
func (TorrentUtil) ClearTorrents(c TorrentConfig) {
	debugx.Println("torrent data directory", c.ClientConfig.DataDir)
	deploysDir := filepath.Join(filepath.Dir(c.ClientConfig.DataDir), bw.DirDeploys)
	dropped := 0
	for _, tt := range c.client.Torrents() {
		if info := tt.Info(); info != nil {
			for _, tf := range tt.Files() {
				deployDir := filepath.Join(deploysDir, tf.Path())
				if _, cause := os.Stat(deployDir); os.IsNotExist(cause) {
					c.client.Stop(tt.Metadata())
					logx.MaybeLog(os.Remove(filepath.Join(c.ClientConfig.DataDir, tf.Path())))
					dropped = dropped + 1
				}
			}
		}
	}

	log.Println(dropped, "torrents dropped due to missing deploy directory")
}

// PrintTorrentInfo prints information about the torrent cluster.
func (t TorrentUtil) PrintTorrentInfo(c TorrentConfig) {
	t.printTorrentInfo(c.client)
}

func (TorrentUtil) printTorrentInfo(c *torrent.Client) {
	c.WriteStatus(os.Stderr)
	log.Println(len(c.Torrents()), "torrents running")
}

func (t TorrentUtil) loadDir(dir string, c *torrent.Client) error {
	return filepath.Walk(dir, func(path string, info os.FileInfo, err error) error {
		const (
			piecesDB = ".torrent.bolt.db"
		)

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

func (TorrentUtil) debugDHT() dht.ServerConfig {
	return dht.ServerConfig{
		OnQuery: func(query *krpc.Msg, source net.Addr) bool {
			log.Println("query", source.String(), spew.Sdump(query))
			return true
		},
		OnAnnouncePeer: func(infoHash metainfo.Hash, ip net.IP, port int, portOK bool) {
			log.Printf("announce peer %s:%d %t %s\n", ip.String(), port, portOK, infoHash.String())
		},
	}
}

func (TorrentUtil) filePath(dir, name string) string {
	return filepath.Join(dir, name)
}

func (t TorrentUtil) createTorrent(dir, name string) (*os.File, error) {
	return t.createFile(t.filePath(dir, name))
}

func (TorrentUtil) createFile(dir string) (*os.File, error) {
	var (
		err error
		dst *os.File
	)

	if err = os.MkdirAll(filepath.Join(dir), 0755); err != nil {
		return nil, errors.WithStack(err)
	}

	if dst, err = ioutil.TempFile(dir, "upload-*.bin"); err != nil {
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
	return metainfo.Magnet{
		DisplayName: meta.DisplayName,
		InfoHash:    meta.InfoHash,
	}
}
