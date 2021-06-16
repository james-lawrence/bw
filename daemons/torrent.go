package daemons

import (
	"net"
	"path/filepath"

	"github.com/james-lawrence/bw"
	"github.com/james-lawrence/bw/internal/torrentx"
	"github.com/james-lawrence/bw/internal/x/tlsx"
	"github.com/james-lawrence/bw/muxer"
	"github.com/james-lawrence/bw/storage"
	"github.com/james-lawrence/torrent"
	"github.com/pkg/errors"
)

// Torrent daemon - used for transferring deploy archives between agents
func Torrent(dctx Context) (tc storage.TorrentConfig, err error) {
	var (
		bind net.Listener
	)

	if bind, err = dctx.Muxer.Bind(bw.ProtocolTorrent, dctx.Listener.Addr()); err != nil {
		return tc, errors.Wrap(err, "failed to bind bw.torrent service")
	}

	b := torrentx.NewMultibind(
		torrent.NewSocketsBind(
			torrentx.Socket{
				Listener: bind,
				Dialer: muxer.NewDialer(
					bw.ProtocolTorrent,
					tlsx.NewDialer(tlsx.MustClone(dctx.RPCCredentials, tlsx.OptionNoClientCert)),
				),
			},
		),
	)

	opts := []storage.TorrentOption{
		storage.TorrentOptionBind(b),
		storage.TorrentOptionDHTPeers(dctx.Cluster),
		storage.TorrentOptionDataDir(filepath.Join(dctx.Config.Root, bw.DirTorrents)),
	}

	return storage.NewTorrent(dctx.Cluster, opts...)
}
