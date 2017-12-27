package storage

import (
	"hash"
	"io"
	"log"
	"os"
	"path/filepath"
	"time"
	// "fmt"

	"github.com/anacrolix/torrent"
	"github.com/anacrolix/torrent/bencode"
	"github.com/anacrolix/torrent/metainfo"
	"github.com/anacrolix/torrent/storage"
	"github.com/davecgh/go-spew/spew"
	"github.com/james-lawrence/bw"
	"github.com/pkg/errors"
)

type torrentU struct {
	uid    bw.RandomID
	sha    hash.Hash
	dst    *os.File
	client *torrent.Client
}

func (t torrentU) Upload(r io.Reader) (hash.Hash, error) {
	return upload(r, t.sha, t.dst)
}

// Info ...
func (t torrentU) Info() (hash.Hash, string, error) {
	var (
		err  error
		newt bool
		mi   metainfo.MetaInfo
		tt   *torrent.Torrent
	)

	if err = t.dst.Sync(); err != nil {
		return nil, "", errors.Wrap(err, "failed to sync to disk")
	}

	if err = t.dst.Close(); err != nil {
		return nil, "", errors.Wrap(err, "failed to close upload")
	}

	if mi, err = torrentInfoFromFile(t.dst.Name()); err != nil {
		return nil, "", err
	}

	log.Println("adding torrent", mi.HashInfoBytes().String(), t.dst.Name())
	spec := torrent.TorrentSpecFromMetaInfo(&mi)
	spec.Storage = storage.NewFile(filepath.Dir(t.dst.Name()))
	if tt, newt, err = t.client.AddTorrentSpec(spec); err != nil {
		return nil, "", errors.WithStack(err)
	} else if newt {
		t.client.DHT().Announce(mi.HashInfoBytes(), 0, true)
		select {
		case <-tt.GotInfo():
			info := tt.Info()
			log.Println("new torrent added", info.Name, info.NumPieces(), spew.Sdump(tt.Stats()))
		case <-time.After(10 * time.Second):
			return nil, "", errors.New("failed to retrieve info")
		}
	}

	return t.sha, mi.Magnet(t.uid.String(), mi.HashInfoBytes()).String(), nil
}

func torrentInfoFromFile(path string) (mi metainfo.MetaInfo, err error) {
	var (
		b []byte
	)

	info := metainfo.Info{PieceLength: 256 * 1024}

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
