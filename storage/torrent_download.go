package storage

import (
	"io"
	"os"
	"time"

	"github.com/anacrolix/torrent"
	"github.com/anacrolix/torrent/metainfo"
	"github.com/anacrolix/torrent/storage"
	"github.com/pkg/errors"
)

type torrentD struct {
	client *torrent.Client
	dir    string
	magnet string
	util   torrentUtil
}

func (t torrentD) Download() io.ReadCloser {
	var (
		err  error
		newt bool
		dst  *os.File
		tt   *torrent.Torrent
		m    metainfo.Magnet
	)

	if m, err = metainfo.ParseMagnetURI(t.magnet); err != nil {
		return newErrReader(errors.WithStack(err))
	}

	if dst, err = t.util.CreateTorrent(t.dir, m.DisplayName); err != nil {
		return newErrReader(errors.WithStack(err))
	}

	fpath := t.util.Dir(t.dir, m.DisplayName)
	peers := make([]torrent.Peer, 0, t.client.DHT().NumNodes())
	for _, n := range t.client.DHT().Nodes() {
		peers = append(peers, torrent.Peer{
			Id:   n.ID,
			IP:   n.Addr.IP,
			Port: n.Addr.Port,
		})
	}

	if tt, newt = t.client.AddTorrentInfoHashWithStorage(m.InfoHash, storage.NewFile(fpath)); newt {
		select {
		case <-tt.GotInfo():
		case <-time.After(30 * time.Second):
			return newErrReader(errors.New("timed out waiting for torrent info"))
		}
		t.client.DHT().Announce(m.InfoHash, 0, true)
	}

	tt.AddPeers(peers)
	tt.DownloadAll()

	// TODO: see about removing the tee reader. does the torrent do the proper thing with file storage.
	return newTeeReader(tt.NewReader(), dst)
}

// TeeReader returns a Reader that writes to w what it reads from r.
// All reads from r performed through it are matched with
// corresponding writes to w. There is no internal buffering -
// the write must complete before the read completes.
// Any error encountered while writing is reported as a read error.
func newTeeReader(t *torrent.Reader, w io.WriteCloser) io.ReadCloser {
	return &teeReader{t, w}
}

type teeReader struct {
	r *torrent.Reader
	w io.WriteCloser
}

func (t *teeReader) Read(p []byte) (n int, err error) {
	n, err = t.r.Read(p)
	if n > 0 {
		if n, err = t.w.Write(p[:n]); err != nil {
			return n, err
		}
	}
	return
}

func (t *teeReader) Close() (err error) {
	if err = t.w.Close(); err != nil {
		t.r.Close()
		return err
	}
	return t.r.Close()
}
