package daemons

import (
	"context"
	"crypto/tls"
	"net"
	"time"

	"github.com/james-lawrence/bw"
	"github.com/james-lawrence/bw/agent"
	"github.com/james-lawrence/bw/agent/dialers"
	"github.com/james-lawrence/bw/agent/discovery"
	"github.com/james-lawrence/bw/clustering"
	"github.com/james-lawrence/bw/internal/x/envx"
	"github.com/james-lawrence/bw/internal/x/logx"
	"github.com/james-lawrence/bw/internal/x/tlsx"
	"github.com/james-lawrence/bw/muxer"
	"github.com/james-lawrence/bw/notary"

	"github.com/hashicorp/memberlist"
	"github.com/pkg/errors"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials"
)

// ConnectOption - options for connecting to the cluster.
type ConnectOption func(*connection)

// ConnectOptionClustering set clustering options for connect.
func ConnectOptionClustering(options ...clustering.Option) ConnectOption {
	return func(c *connection) {
		c.clustering.Options = options
	}
}

// ConnectOptionBootstrap set clustering options for bootstrap.
func ConnectOptionBootstrap(options ...clustering.BootstrapOption) ConnectOption {
	return func(c *connection) {
		c.clustering.Bootstrap = options
	}
}

type connection struct {
	clustering struct {
		Options   []clustering.Option
		Bootstrap []clustering.BootstrapOption
		Snapshot  []clustering.SnapshotOption
	}
}

// Connect returning just a single client to the caller.
func Connect(config agent.ConfigClient, ss notary.Signer, options ...grpc.DialOption) (d dialers.Direct, c clustering.C, err error) {
	var (
		creds credentials.TransportCredentials
	)

	if creds, err = GRPCGenClient(config); err != nil {
		return d, c, err
	}

	return connect(config, ss, creds, options...)
}

// ConnectClientUntilSuccess continuously tries to make a connection until successful.
func ConnectClientUntilSuccess(
	ctx context.Context,
	config agent.ConfigClient,
	ss notary.Signer,
	options ...grpc.DialOption,
) (d dialers.Direct, c clustering.LocalRendezvous, err error) {
	var (
		creds credentials.TransportCredentials
	)

	if creds, err = GRPCGenClient(config); err != nil {
		return d, c, err
	}

	for i := 0; ; i++ {
		if d, c, err = connect(config, ss, creds, options...); err == nil {
			return d, c, err
		}

		select {
		case <-ctx.Done():
			return d, c, ctx.Err()
		default:
		}

		logx.MaybeLog(errors.Wrap(err, "connection failed"))

		time.Sleep(250 * time.Millisecond)
	}
}

func DefaultDialer(address string, d dialer, options ...grpc.DialOption) (_d dialers.Defaults, err error) {
	var (
		addr *net.TCPAddr
	)

	if addr, err = net.ResolveTCPAddr("tcp", address); err != nil {
		return _d, err
	}

	return dialers.NewDefaults(options...).Defaults(
		dialers.WithMuxer(
			d,
			addr,
		),
		grpc.WithInsecure(),
	), nil
}

// connect discovers the current nodes in the cluster, generating a static cluster for use by the agents to perform work.
func connect(config agent.ConfigClient, ss notary.Signer, creds credentials.TransportCredentials, options ...grpc.DialOption) (d dialers.Direct, c clustering.Static, err error) {
	var (
		dd        dialers.Defaults
		nodes     []*memberlist.Node
		tlsconfig *tls.Config
	)

	if tlsconfig, err = TLSGenClient(config); err != nil {
		return d, c, err
	}

	var di dialer = tlsx.NewDialer(tlsconfig)

	if envx.Boolean(false, bw.EnvProxyDialerEnabled) {
		di = discovery.ProxyDialer{
			Proxy:  config.Address,
			Signer: ss,
			Dialer: muxer.NewDialer(
				bw.ProtocolProxy,
				tlsx.NewDialer(tlsconfig),
			),
		}
	}

	if dd, err = DefaultDialer(
		config.Address,
		di,
		options...,
	); err != nil {
		return d, c, err
	}

	if nodes, err = discovery.Snapshot(agent.DiscoveryP2PAddress(config.Address), dd.Defaults()...); err != nil {
		return d, c, err
	}

	if len(nodes) == 0 {
		return d, c, errors.New("no agents found")
	}

	return dialers.NewDirect(agent.DiscoveryP2PAddress(config.Address), dd.Defaults()...), clustering.NewStatic(nodes...), err
}
