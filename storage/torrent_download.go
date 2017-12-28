package storage

import (
	"io"
	"time"

	"github.com/anacrolix/torrent"
	"github.com/anacrolix/torrent/metainfo"
	"github.com/pkg/errors"
)

type torrentD struct {
	client *torrent.Client
	dir    string
	magnet string
	util   TorrentUtil
}

func (t torrentD) Download() io.ReadCloser {
	var (
		err error
		ok  bool
		tt  *torrent.Torrent
		m   metainfo.Magnet
	)

	if m, err = metainfo.ParseMagnetURI(t.magnet); err != nil {
		return newErrReader(errors.WithStack(err))
	}

	if tt, ok = t.client.AddTorrentInfoHash(m.InfoHash); ok {
		t.client.DHT().Announce(m.InfoHash, 0, true)
	}

	select {
	case <-tt.GotInfo():
	case <-time.After(30 * time.Second):
		return newErrReader(errors.New("timed out waiting for torrent info"))
	}
	tt.DownloadAll()

	return tt.NewReader()
}
