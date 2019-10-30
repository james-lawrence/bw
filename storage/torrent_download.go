package storage

import (
	"context"
	"io"
	"net"
	"time"

	"github.com/anacrolix/dht/v2/krpc"
	"github.com/anacrolix/torrent"
	"github.com/james-lawrence/bw/agent"
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
		err error
		tt  *torrent.Torrent
	)

	wait, cancel := context.WithTimeout(ctx, 30*time.Second)
	defer cancel()

	if tt, err = t.client.AddMagnet(archive.Location); err != nil {
		return newErrReader(errors.WithStack(err))
	}

	select {
	case <-tt.GotInfo():
	case <-wait.Done():
		return newErrReader(errors.New("timed out waiting for torrent info"))
	}

	tt.DownloadAll()

	return tt.NewReader()
}
