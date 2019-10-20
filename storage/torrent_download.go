package storage

import (
	"context"
	"io"
	"log"
	"net"
	"time"

	"github.com/anacrolix/dht/v2/krpc"
	"github.com/anacrolix/torrent"
	"github.com/anacrolix/torrent/metainfo"
	"github.com/james-lawrence/bw/agent"
	"github.com/james-lawrence/bw/internal/x/logx"
	"github.com/pkg/errors"
)

type torrentD struct {
	c      cluster
	client *torrent.Client
	dir    string
	util   TorrentUtil
}

func peerToNode(p agent.Peer) krpc.NodeInfo {
	udp := &net.UDPAddr{IP: net.ParseIP(p.Ip), Port: int(p.TorrentPort)}
	na := krpc.NodeAddr{}
	na.FromUDPAddr(udp)

	return krpc.NodeInfo{
		Addr: na,
	}
}

func peersToNode(peers ...agent.Peer) (r []krpc.NodeInfo) {
	for _, p := range peers {
		r = append(r, peerToNode(p))
	}
	return r
}

func (t torrentD) Download(ctx context.Context, archive agent.Archive) io.ReadCloser {
	var (
		err        error
		dlrequired bool
		tt         *torrent.Torrent
		m          metainfo.Magnet
	)

	for _, s := range t.client.DhtServers() {
		n := peerToNode(*archive.Peer)
		if err = s.AddNode(n); err != nil {
			return newErrReader(errors.WithStack(err))
		}

		for _, n := range peersToNode(t.c.Quorum()...) {
			log.Println("adding peer to DHT", n.Addr.String())
			if err = s.AddNode(n); err != nil {
				return newErrReader(errors.WithStack(err))
			}
		}
		s.Bootstrap()
	}

	if m, err = metainfo.ParseMagnetURI(archive.Location); err != nil {
		return newErrReader(errors.WithStack(err))
	}

	wait, cancel := context.WithTimeout(ctx, 30*time.Second)
	defer cancel()

	if tt, dlrequired = t.client.AddTorrentInfoHash(m.InfoHash); dlrequired {
		for _, s := range t.client.DhtServers() {
			_, err := s.Announce(m.InfoHash, 0, true)
			logx.MaybeLog(errors.Wrap(err, "announce failed"))
		}

		select {
		case <-tt.GotInfo():
		case <-wait.Done():
			return newErrReader(errors.New("timed out waiting for torrent info"))
		}

		tt.DownloadAll()
	}

	if tt == nil {
		log.Println("missing torrent", dlrequired)
		return newErrReader(errors.New("failed to successfully add infohash"))
	}

	return tt.NewReader()
}
