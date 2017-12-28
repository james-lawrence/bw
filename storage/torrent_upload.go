package storage

import (
	"hash"
	"io"
	"os"
	// "fmt"

	"github.com/anacrolix/torrent"
	"github.com/anacrolix/torrent/metainfo"
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
		mi   metainfo.MetaInfo
		util TorrentUtil
	)

	if err = t.dst.Sync(); err != nil {
		return nil, "", errors.Wrap(err, "failed to sync to disk")
	}

	if err = t.dst.Close(); err != nil {
		return nil, "", errors.Wrap(err, "failed to close upload")
	}

	if mi, err = util.loadTorrent(t.client, t.dst.Name()); err != nil {
		return nil, "", err
	}

	return t.sha, mi.Magnet(t.uid.String(), mi.HashInfoBytes()).String(), nil
}
