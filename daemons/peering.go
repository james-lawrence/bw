package daemons

import (
	"context"
	"net"
	"path/filepath"

	"github.com/hashicorp/memberlist"
	"github.com/james-lawrence/bw"
	"github.com/james-lawrence/bw/agent"
	_cluster "github.com/james-lawrence/bw/cluster"
	"github.com/james-lawrence/bw/clustering"
	"github.com/james-lawrence/bw/clustering/peering"
	"github.com/james-lawrence/bw/internal/memberlistx"
	"github.com/james-lawrence/bw/internal/x/tlsx"
	"github.com/james-lawrence/bw/muxer"
	"github.com/pkg/errors"
)

type connecter interface {
	Join(ctx context.Context, conf agent.Config, c clustering.Joiner, snap peering.File) error
	Snapshot(c clustering.Rendezvous, fssnapshot peering.File, options ...clustering.SnapshotOption)
}

// NewClusterDialer dial a cluster based on the configuration.
func NewClusterDialer(conf agent.Config, options ...clustering.Option) clustering.Dialer {
	options = append(
		[]clustering.Option{
			clustering.OptionEventDelegate(_cluster.LoggingEventHandler{}),
			clustering.OptionAliveDelegate(_cluster.AliveDefault{}),
		},
		options...,
	)

	return clustering.NewDialer(options...)
}

// Peering establish peering protocol.
func Peering(dctx Context) (_ Context, err error) {
	var (
		bindreliable net.Listener
		bindpacket   net.PacketConn
		c            clustering.Memberlist
		keyring      *memberlist.Keyring
	)

	if keyring, err = dctx.Config.Keyring(); err != nil {
		return dctx, errors.Wrap(err, "failed to build keyring")
	}

	if bindreliable, err = dctx.Muxer.Bind(bw.ProtocolSWIM, dctx.Listener.Addr()); err != nil {
		return dctx, errors.Wrap(err, "failed to establish reliable transport")
	}

	if bindpacket, err = net.ListenUDP("udp", &net.UDPAddr{IP: dctx.Config.P2PBind.IP, Port: dctx.Config.P2PBind.Port}); err != nil {
		return dctx, errors.Wrap(err, "failed to establish udp transport")
	}

	// TLS verification doesn't matter for swim, since we use a secret key but we need to still
	// pass through the TLS handshake.
	transport, err := memberlistx.NewSWIMTransport(
		muxer.NewDialer(bw.ProtocolSWIM, tlsx.NewDialer(tlsx.MustClone(dctx.RPCCredentials, tlsx.OptionInsecureSkipVerify, tlsx.OptionNoClientCert))),
		memberlistx.SWIMStreams(bindreliable),
		memberlistx.SWIMPackets(bindpacket),
	)

	if err != nil {
		return dctx, errors.Wrap(err, "failed to build swim transport")
	}

	cdialer := NewClusterDialer(
		dctx.Config,
		clustering.OptionNodeID(dctx.Local.Peer.Name),
		clustering.OptionAdvertiseAddress(dctx.AdvertisedIP),
		clustering.OptionAdvertisePort(int(dctx.Local.Peer.P2PPort)),
		clustering.OptionDelegate(dctx.PeeringEvents),
		clustering.OptionKeyring(keyring),
		clustering.OptionEventDelegate(dctx.PeeringEvents),
		clustering.OptionAliveDelegate(_cluster.AliveDefault{}),
		clustering.OptionLogger(dctx.DebugLog),
		clustering.OptionTransport(transport),
	)

	if c, err = cdialer.Dial(); err != nil {
		return dctx, errors.Wrap(err, "failed to create cluster")
	}

	dctx.Cluster = _cluster.New(dctx.Local, c)
	dctx.Bootstrapper = c
	return dctx, err
}

// Peered establish peering.
func Peered(dctx Context, cc connecter) (_ Context, err error) {
	fssnapshot := peering.File{
		Path: filepath.Join(dctx.Config.Root, "cluster.snapshot"),
	}

	if err = cc.Join(dctx.Context, dctx.Config, dctx.Bootstrapper, fssnapshot); err != nil {
		return dctx, errors.Wrap(err, "failed to join cluster")
	}

	cc.Snapshot(
		dctx.Cluster,
		fssnapshot,
		clustering.SnapshotOptionFrequency(dctx.Config.SnapshotFrequency),
		clustering.SnapshotOptionContext(dctx.Context),
	)

	return dctx, err
}
