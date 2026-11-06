package daemons

import (
	"net"
	"path/filepath"

	"github.com/james-lawrence/bw"
	"github.com/james-lawrence/bw/internal/tlsx"
	"github.com/james-lawrence/bw/internal/torrentx"
	"github.com/james-lawrence/bw/muxer"
	"github.com/james-lawrence/bw/storage"
	"github.com/james-lawrence/torrent"
	"github.com/pkg/errors"
)

// Torrent daemon - used for transferring deploy archives between agents
func Torrent(ctx Context) (tc storage.TorrentConfig, err error) {
	var (
		bind net.Listener
	)

	if bind, err = ctx.Muxer.Bind(bw.ProtocolTorrent, ctx.Listener.Addr()); err != nil {
		return tc, errors.Wrap(err, "failed to bind bw.torrent service")
	}

	b := torrent.NewSocketsBind(
		torrentx.Socket{
			Listener: bind,
			// prefer quic, peers without it (disabled, older versions, udp blocked) are reached over tcp.
			Dialer: muxer.NewDialer(
				bw.ProtocolTorrent,
				ctx.QUICDialer(tlsx.NewDialer(tlsx.MustClone(ctx.RPCCredentials, tlsx.OptionInsecureSkipVerify, tlsx.OptionNoClientCert))),
			),
		},
	)

	opts := []storage.TorrentOption{
		storage.TorrentOptionBind(b),
		storage.TorrentOptionDataDir(filepath.Join(ctx.Config.Root, bw.DirTorrents)),
	}

	return storage.NewTorrent(ctx.Cluster, opts...)
}
