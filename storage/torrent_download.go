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
	stall  time.Duration // window without progress before restarting.
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
	metadata, err := torrent.NewFromMagnet(archive.Location)
	if err != nil {
		return newErrReader(errors.WithStack(err))
	}

	pr, pw := io.Pipe()

	go func() {
		var (
			pos   int64
			stall = t.stall
		)

		// restart stalled downloads until the context is cancelled.
		for ctx.Err() == nil {
			var dl torrent.Torrent

			if dl, _, err = t.client.Start(metadata, torrent.TunePeers(peersToNode(t.c.Quorum()...)...)); err != nil {
				break
			}

			// no metadata within the window is a stall as well (e.g. no reachable peers).
			mctx, cancel := context.WithTimeoutCause(ctx, stall, ErrDownloadStalled)
			rc := torrent.Downloader(mctx, dl)
			// the reader returned on timeout wraps the cause without Unwrap, so errors.Is can't see it.
			metadataStalled := ctx.Err() == nil && errors.Is(context.Cause(mctx), ErrDownloadStalled)
			cancel()

			// a restarted torrent replays from the start, continue from what the consumer already has.
			if _, err = rc.Seek(pos, io.SeekStart); err == nil {
				var n int64

				// fail the attempt when no progress has been made, replaces a fixed timeout.
				sr := iox.NewStalledReader(rc, stall, ErrDownloadStalled, nil)
				stop := context.AfterFunc(ctx, func() { sr.Close() })
				n, err = io.Copy(pw, sr)
				stop()
				sr.Close()
				pos += n
			} else if metadataStalled {
				err = ErrDownloadStalled
			}

			if !errors.Is(err, ErrDownloadStalled) {
				break
			}

			errorsx.Log(errors.Wrap(t.client.Stop(metadata), "unable to stop stalled torrent"))
		}

		if ctx.Err() != nil {
			err = context.Cause(ctx)
		}

		pw.CloseWithError(err)
	}()

	return pr
}
