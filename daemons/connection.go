package daemons

import (
	"context"
	"crypto/tls"
	"log"
	"time"

	"github.com/james-lawrence/bw"
	"github.com/james-lawrence/bw/agent"
	"github.com/james-lawrence/bw/agent/dialers"
	"github.com/james-lawrence/bw/agent/discovery"
	"github.com/james-lawrence/bw/certificatecache"
	"github.com/james-lawrence/bw/clustering"
	"github.com/james-lawrence/bw/internal/errorsx"
	"github.com/james-lawrence/bw/internal/grpcx"
	"github.com/james-lawrence/bw/internal/tlsx"
	"github.com/james-lawrence/bw/muxer"
	"github.com/james-lawrence/bw/notary"

	"github.com/hashicorp/memberlist"
	"github.com/pkg/errors"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
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
func Connect(ctx context.Context, config agent.ConfigClient, ss notary.Signer, options ...grpc.DialOption) (d dialers.Direct, c clustering.Rendezvous, err error) {
	return connect(ctx, config, ss, options...)
}

// ConnectClientUntilSuccess continuously tries to make a connection until successful.
func ConnectClientUntilSuccess(
	ctx context.Context,
	config agent.ConfigClient,
	ss notary.Signer,
	options ...grpc.DialOption,
) (d dialers.Direct, c clustering.Rendezvous, err error) {
	for i := 0; ; i++ {
		if d, c, err = connect(ctx, config, ss, options...); err == nil {
			return d, c, err
		}

		select {
		case <-ctx.Done():
			return d, c, ctx.Err()
		default:
		}

		errorsx.Log(errors.Wrap(err, "connection failed"))

		time.Sleep(250 * time.Millisecond)
	}
}

// connect discovers the current nodes in the cluster, generating a static cluster for use by the agents to perform work.
func connect(ctx context.Context, config agent.ConfigClient, ss notary.Signer, options ...grpc.DialOption) (d dialers.Direct, c clustering.Rendezvous, err error) {
	var (
		dd        dialers.Defaults
		nodes     []*memberlist.Node
		tlsconfig *tls.Config
	)

	if tlsconfig, err = certificatecache.TLSGenClient(config); err != nil {
		return d, c, err
	}

	var di dialer = discovery.ProxyDialer{
		Proxy:  config.Address,
		Signer: ss,
		Dialer: muxer.NewDialer(
			bw.ProtocolProxy,
			tlsx.NewDialer(tlsconfig),
		),
	}

	if dd, err = dialers.DefaultDialer(
		config.Address,
		di,
		options...,
	); err != nil {
		return d, c, err
	}

	c = clustering.NewCached(func(ctx context.Context) clustering.Rendezvous {
		err = grpcx.Retry(ctx, func() error {
			if nodes, err = discovery.Snapshot(agent.URIDiscovery(config.Address), dd.Defaults()...); err != nil {
				return err
			}

			return nil
		}, codes.Unavailable)

		if err != nil {
			log.Println("snapshot failed", err)
			return clustering.NewStatic()
		}

		return clustering.NewStatic(nodes...)
	})

	if len(c.Members()) == 0 {
		return d, c, errors.New("no agents found")
	}

	return dialers.NewDirect(agent.URIDiscovery(config.Address), dd.Defaults()...), c, err
}
