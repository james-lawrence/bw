package storage

import (
	"log"
	"net"
	"os"
	"path/filepath"

	"github.com/anacrolix/dht"
	"github.com/anacrolix/dht/krpc"
	"github.com/anacrolix/torrent/metainfo"
	"github.com/davecgh/go-spew/spew"
	"github.com/pkg/errors"
)

type torrentUtil struct{}

func (torrentUtil) debugDHT() dht.ServerConfig {
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

func (torrentUtil) Dir(dir, name string) string {
	return filepath.Join(dir, name, ".torrent")
}

func (torrentUtil) FilePath(dir, name string) string {
	return filepath.Join(dir, name, ".torrent", name)
}

func (t torrentUtil) CreateTorrent(dir, name string) (*os.File, error) {
	return t.CreateFile(t.FilePath(dir, name))
}

func (torrentUtil) CreateFile(path string) (*os.File, error) {
	var (
		err error
		dst *os.File
	)

	if err = os.MkdirAll(filepath.Join(filepath.Dir(path)), 0755); err != nil {
		return nil, errors.WithStack(err)
	}

	if dst, err = os.Create(path); err != nil {
		return nil, errors.WithStack(err)
	}

	return dst, err
}
