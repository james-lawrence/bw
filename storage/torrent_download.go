package storage

import (
	"context"
	"io"
	"log"
	"net"

	"github.com/anacrolix/dht/v2"
	"github.com/anacrolix/dht/v2/krpc"
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
	udp := &net.UDPAddr{IP: net.ParseIP(archive.Peer.Ip), Port: int(archive.Peer.TorrentPort)}
	na := krpc.NodeAddr{}
	na.FromUDPAddr(udp)

	ni := krpc.NodeInfo{
		ID:   dht.MakeDeterministicNodeID(udp),
		Addr: na,
	}

	log.Println("adding peer to DHT", ni.Addr.String())
	for _, s := range t.client.DhtServers() {
		if err = s.AddNode(ni); err != nil {
			return newErrReader(errors.WithStack(err))
		}

		if stats, err = s.Bootstrap(); err != nil {
			return newErrReader(errors.WithStack(err))
		}

		log.Println("dht bootstrap stats", spew.Sdump(stats))
	}

	if m, err = metainfo.ParseMagnetURI(archive.Location); err != nil {
		return newErrReader(errors.WithStack(err))
	}

	if tt, ok = t.client.AddTorrentInfoHash(m.InfoHash); ok {
		for _, s := range t.client.DhtServers() {
			s.Announce(m.InfoHash, 0, true)
		}
	}

	select {
	case <-tt.GotInfo():
	case <-ctx.Done():
		return newErrReader(errors.New("timed out waiting for torrent info"))
	}

	tt.DownloadAll()

	return tt.NewReader()
}
