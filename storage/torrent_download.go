package storage

import (
	"context"
	"crypto/sha1"
	"io"
	"net/netip"
	"time"

	"github.com/james-lawrence/bw/agent"
	"github.com/james-lawrence/bw/internal/errorsx"
	"github.com/james-lawrence/bw/internal/iox"
	"github.com/james-lawrence/torrent"
	"github.com/james-lawrence/torrent/dht/int160"
	"github.com/pkg/errors"
)

const ErrDownloadStalled = errorsx.String("download stalled: no progress")

type torrentD struct {
	c      cluster
	client *torrent.Client
	dir    string
}

// the client has no dht server, so it considers the zero id to be itself and refuses to dial any
// peer using it. we don't know the remote's real id so derive a stable non-zero one from its address.
func peerID(addr netip.AddrPort) int160.T {
	digest := sha1.Sum([]byte(addr.String()))
	return int160.FromByteArray(digest)
}

func peerToNode(p *agent.Peer) (torrent.Peer, bool) {
	ip, err := netip.ParseAddr(p.Ip)
	if err != nil {
		return torrent.Peer{}, false
	}

	addr := netip.AddrPortFrom(ip, uint16(p.P2PPort))

	return torrent.NewPeer(
		peerID(addr),
		addr,
		torrent.PeerOptionSource("X"),
		torrent.PeerOptionTrusted(true),
	), true
}

func peersToNode(peers ...*agent.Peer) (r []torrent.Peer) {
	for _, p := range peers {
		if n, ok := peerToNode(p); ok {
			r = append(r, n)
		}
	}
	return r
}

func (t torrentD) Download(ctx context.Context, archive *agent.Archive) (r io.ReadCloser) {
	var (
		err      error
		metadata torrent.Metadata
	)

	if metadata, err = torrent.NewFromMagnet(archive.Location); err != nil {
		return newErrReader(errors.WithStack(err))
	}

	dl, _, err := t.client.Start(metadata, torrent.TunePeers(peersToNode(t.c.Quorum()...)...))
	if err != nil {
		return newErrReader(errors.WithStack(err))
	}

	pr, pw := io.Pipe()

	go func() {
		// fail the download when no progress has been made, replaces a fixed timeout.
		ctx, cancel := context.WithCancelCause(ctx)
		w := iox.NewStallWriter(pw, 16*time.Second, ErrDownloadStalled, cancel)
		w.CloseWithError(errorsx.From(torrent.DownloadInto(ctx, w, dl)))
	}()

	return pr
}
