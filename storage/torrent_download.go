package storage

import (
	"io"
	"log"
	"os"
	"path/filepath"
	"time"

	"github.com/anacrolix/torrent"
	"github.com/anacrolix/torrent/storage"
	"github.com/davecgh/go-spew/spew"
	"github.com/pkg/errors"
)

type torrentD struct {
	client *torrent.Client
	dir    string
	magnet string
}

func (t torrentD) Download() io.ReadCloser {
	var (
		err  error
		newt bool
		dst  *os.File
		tt   *torrent.Torrent
		spec *torrent.TorrentSpec
	)

	peers := make([]torrent.Peer, 0, t.client.DHT().NumNodes())
	for _, n := range t.client.DHT().Nodes() {
		peers = append(peers, torrent.Peer{
			Id:   n.ID,
			IP:   n.Addr.IP,
			Port: n.Addr.Port,
		})
	}

	if spec, err = torrent.TorrentSpecFromMagnetURI(t.magnet); err != nil {
		return newErrReader(errors.WithStack(err))
	}
	fpath := filepath.Join(t.dir, spec.DisplayName, ".torrent")
	spec.Storage = storage.NewFile(fpath)

	if tt, newt, err = t.client.AddTorrentSpec(spec); err != nil {
		return newErrReader(errors.WithStack(err))
	} else if newt {
		timeout := time.After(30 * time.Second)
		select {
		case <-tt.GotInfo():
		case <-timeout:
			return newErrReader(errors.New("timed out waiting for torrent info"))
		}

		log.Println("KNOWN PEERS", spew.Sdump(peers))
		tt.AddPeers(peers)
	}

	info := tt.Info()
	log.Println("DOWNLOADING", fpath, info.Length)
	tt.DownloadAll()
	t.client.DHT().Announce(tt.InfoHash(), 0, true)

	if dst, err = torrentDestination(filepath.Join(t.dir, info.Name, ".torrent", info.Name)); err != nil {
		return newErrReader(errors.WithStack(err))
	}

	t.client.DHT().Announce(tt.InfoHash(), 0, true)
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
	log.Println("TRYING TO READ FILE")
	// ctx := context.WithTimeout(context.Background(), 5*time.Second)
	n, err = t.r.Read(p)
	log.Println("READ FILE BYTES", n, err)
	if n > 0 {
		if n, err := t.w.Write(p[:n]); err != nil {
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
