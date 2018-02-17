package storage

import (
	"context"
	"io"
	"log"
	"net"

	"github.com/anacrolix/dht"
	"github.com/anacrolix/dht/krpc"
	"github.com/anacrolix/torrent"
	"github.com/anacrolix/torrent/metainfo"
	"github.com/davecgh/go-spew/spew"
	"github.com/james-lawrence/bw/agent"
	"github.com/pkg/errors"
)

type torrentD struct {
	client *torrent.Client
	dir    string
	util   TorrentUtil
}

func (t torrentD) Download(ctx context.Context, archive agent.Archive) io.ReadCloser {
	var (
		err   error
		ok    bool
		tt    *torrent.Torrent
		m     metainfo.Magnet
		stats dht.TraversalStats
	)

	ni := krpc.NodeInfo{Addr: &net.UDPAddr{IP: net.ParseIP(archive.Peer.Ip), Port: int(archive.Peer.TorrentPort)}}

	if stats, err = t.client.DHT().Bootstrap(); err != nil {
		return newErrReader(errors.WithStack(err))
	}

	log.Println("adding peer to DHT", ni.Addr.String(), spew.Sdump(stats))
	if err = t.client.DHT().AddNode(ni); err != nil {
		return newErrReader(errors.WithStack(err))
	}

	t.util.printTorrentInfo(t.client)

	if m, err = metainfo.ParseMagnetURI(archive.Location); err != nil {
		return newErrReader(errors.WithStack(err))
	}

	if tt, ok = t.client.AddTorrentInfoHash(m.InfoHash); ok {
		t.client.DHT().Announce(m.InfoHash, 0, true)
	}

	select {
	case <-tt.GotInfo():
	case <-ctx.Done():
		return newErrReader(errors.New("timed out waiting for torrent info"))
	}

	tt.DownloadAll()

	return tt.NewReader()
}
