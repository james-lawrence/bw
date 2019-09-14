package storage

import (
	"context"
	"io"
	"log"
	"net"

	"github.com/anacrolix/dht/v2/krpc"
	"github.com/anacrolix/torrent"
	"github.com/anacrolix/torrent/metainfo"
	"github.com/james-lawrence/bw/agent"
	"github.com/james-lawrence/bw/internal/x/logx"
	"github.com/pkg/errors"
)

type torrentD struct {
	client *torrent.Client
	dir    string
	util   TorrentUtil
}

func (t torrentD) Download(ctx context.Context, archive agent.Archive) io.ReadCloser {
	var (
		err error
		ok  bool
		tt  *torrent.Torrent
		m   metainfo.Magnet
	)
	udp := &net.UDPAddr{IP: net.ParseIP(archive.Peer.Ip), Port: int(archive.Peer.TorrentPort)}
	na := krpc.NodeAddr{}
	na.FromUDPAddr(udp)

	ni := krpc.NodeInfo{
		Addr: na,
	}

	log.Println("adding peer to DHT", ni.Addr.String())
	for _, s := range t.client.DhtServers() {
		if err = s.AddNode(ni); err != nil {
			return newErrReader(errors.WithStack(err))
		}
		s.Bootstrap()
	}

	if m, err = metainfo.ParseMagnetURI(archive.Location); err != nil {
		return newErrReader(errors.WithStack(err))
	}

	if tt, ok = t.client.AddTorrentInfoHash(m.InfoHash); ok {
		for _, s := range t.client.DhtServers() {
			_, err := s.Announce(m.InfoHash, 0, true)
			logx.MaybeLog(errors.Wrap(err, "announce failed"))
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
