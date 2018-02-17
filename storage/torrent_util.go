package storage

import (
	"log"
	"net"
	"os"
	"path/filepath"

	"github.com/anacrolix/dht"
	"github.com/anacrolix/dht/krpc"
	"github.com/anacrolix/missinggo"
	"github.com/anacrolix/torrent"
	"github.com/anacrolix/torrent/bencode"
	"github.com/anacrolix/torrent/metainfo"
	"github.com/davecgh/go-spew/spew"
	"github.com/james-lawrence/bw"
	"github.com/james-lawrence/bw/x/debugx"
	"github.com/james-lawrence/bw/x/logx"
	"github.com/pkg/errors"
)

// TorrentUtil utility functions for the torrent storage subsystem.
type TorrentUtil struct{}

// ClearTorrents periodically flushes torrents from storage based on whether or not
// the deployment directory is still around.
func (TorrentUtil) ClearTorrents(c TorrentConfig) {
	debugx.Println("torrent data directory", c.Config.DataDir)
	deploysDir := filepath.Join(filepath.Dir(c.Config.DataDir), bw.DirDeploys)
	dropped := 0
	for _, tt := range c.client.Torrents() {
		if info := tt.Info(); info != nil {
			for _, tf := range tt.Files() {
				deployDir := filepath.Join(deploysDir, tf.Path())
				if _, cause := os.Stat(deployDir); os.IsNotExist(cause) {
					tt.Drop()
					logx.MaybeLog(os.Remove(filepath.Join(c.Config.DataDir, tf.Path())))
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
	c.DHT().WriteStatus(os.Stderr)
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
		OnAnnouncePeer: func(infoHash metainfo.Hash, peer dht.Peer) {
			log.Println("announce peer", peer.String(), infoHash.String())
		},
	}
}

func (TorrentUtil) filePath(dir, name string) string {
	return filepath.Join(dir, name)
}

func (t TorrentUtil) createTorrent(dir, name string) (*os.File, error) {
	return t.createFile(t.filePath(dir, name))
}

func (TorrentUtil) createFile(path string) (*os.File, error) {
	var (
		err error
		dst *os.File
	)

	if err = os.MkdirAll(filepath.Join(filepath.Dir(path)), 0755); err != nil {
		return nil, errors.WithStack(err)
	}

	if dst, err = os.OpenFile(path, os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0600); err != nil {
		return nil, errors.WithStack(err)
	}

	return dst, err
}

func (t TorrentUtil) loadTorrent(c *torrent.Client, path string) (mi metainfo.MetaInfo, err error) {
	if mi, err = t.infoFromFile(path); err != nil {
		return mi, err
	}

	ts := torrent.TorrentSpecFromMetaInfo(&mi)
	if _, _, err = c.AddTorrentSpec(ts); err != nil {
		return mi, errors.WithStack(err)
	}

	c.DHT().Announce(mi.HashInfoBytes(), 0, true)

	return mi, nil
}

func (TorrentUtil) infoFromFile(path string) (mi metainfo.MetaInfo, err error) {
	var (
		b []byte
	)

	info := metainfo.Info{PieceLength: missinggo.MiB}

	if err = info.BuildFromFilePath(path); err != nil {
		return mi, errors.WithStack(err)
	}

	if b, err = bencode.Marshal(info); err != nil {
		return mi, errors.WithStack(err)
	}

	mi = metainfo.MetaInfo{
		InfoBytes: b,
	}
	mi.SetDefaults()

	return mi, nil
}
