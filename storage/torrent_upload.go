package storage

import (
	"encoding/hex"
	"hash"
	"io"
	"os"
	"path/filepath"

	"github.com/anacrolix/torrent"
	"github.com/anacrolix/torrent/metainfo"
	"github.com/pkg/errors"
)

type torrentU struct {
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

	uid := hex.EncodeToString(t.sha.Sum(nil))
	path := filepath.Join(filepath.Dir(t.dst.Name()), uid)
	if err = t.dst.Sync(); err != nil {
		return nil, "", errors.Wrap(err, "failed to sync to disk")
	}

	if err = t.dst.Close(); err != nil {
		return nil, "", errors.Wrap(err, "failed to close upload")
	}

	if err = os.Rename(t.dst.Name(), path); err != nil {
		return nil, "", errors.Wrap(err, "failed to rename upload")
	}

	if mi, err = util.loadTorrent(t.client, path); err != nil {
		return nil, "", err
	}

	return t.sha, mi.Magnet(uid, mi.HashInfoBytes()).String(), nil
}
