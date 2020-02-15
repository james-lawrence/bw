package storage

import (
	"context"
	"io"
	"net"
	"time"

	"github.com/james-lawrence/bw/agent"
	"github.com/james-lawrence/torrent"
	"github.com/pkg/errors"
)

type torrentD struct {
	c      cluster
	client *torrent.Client
	dir    string
	util   TorrentUtil
}

func peerToNode(p agent.Peer) torrent.Peer {
	return torrent.Peer{
		IP:      net.ParseIP(p.Ip),
		Port:    int(p.TorrentPort),
		Source:  "X",
		Trusted: true,
	}
}

func peersToNode(peers ...agent.Peer) (r []torrent.Peer) {
	for _, p := range peers {
		r = append(r, peerToNode(p))
	}
	return r
}

func (t torrentD) Download(ctx context.Context, archive agent.Archive) (r io.ReadCloser) {
	var (
		err      error
		metadata torrent.Metadata
	)

	if metadata, err = torrent.NewFromMagnet(archive.Location); err != nil {
		return newErrReader(errors.WithStack(err))
	}

	wait, cancel := context.WithTimeout(ctx, 30*time.Second)
	defer cancel()

	if r, err = t.client.Download(wait, metadata, torrent.TunePeers(peersToNode(t.c.Quorum()...)...)); err != nil {
		return newErrReader(errors.WithStack(err))
	}

	return r
}
