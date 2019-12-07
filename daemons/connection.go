package daemons

import (
	"context"
	"io/ioutil"
	"time"

	"github.com/james-lawrence/bw/agent"
	"github.com/james-lawrence/bw/clustering"
	"github.com/james-lawrence/bw/internal/x/logx"

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

func newConnect(options ...ConnectOption) connection {
	var (
		conn connection
	)

	for _, opt := range options {
		opt(&conn)
	}

	return conn
}

type connection struct {
	clustering struct {
		Options   []clustering.Option
		Bootstrap []clustering.BootstrapOption
		Snapshot  []clustering.SnapshotOption
	}
}

// Connect returning just a single client to the caller.
func Connect(config agent.ConfigClient, options ...ConnectOption) (cl agent.Client, d agent.Dialer, c clustering.Cluster, err error) {
	var (
		creds credentials.TransportCredentials
	)

	if creds, err = GRPCGenClient(config); err != nil {
		return cl, d, c, err
	}

	return connect(config, creds, options...)
}

// ConnectClientUntilSuccess continuously tries to make a connection until successful.
func ConnectClientUntilSuccess(
	ctx context.Context,
	config agent.ConfigClient, onRetry func(error), options ...ConnectOption,
) (client agent.Client, d agent.Dialer, c clustering.Cluster, err error) {
	var (
		creds credentials.TransportCredentials
	)

	if creds, err = GRPCGenClient(config); err != nil {
		return client, d, c, err
	}

	for i := 0; ; i++ {
		if client, d, c, err = connect(config, creds, options...); err == nil {
			return client, d, c, err
		}

		// when an error occurs, cleanup any resources.
		logx.MaybeLog(errors.WithMessage(c.Shutdown(), "failed to cleanup cluster"))
		if client != nil {
			logx.MaybeLog(errors.WithMessage(client.Close(), "failed to cleanup client"))
		}

		select {
		case <-ctx.Done():
			return client, d, c, ctx.Err()
		default:
		}

		onRetry(err)
		time.Sleep(250 * time.Millisecond)
	}
}

func connect(config agent.ConfigClient, creds credentials.TransportCredentials, options ...ConnectOption) (cl agent.Client, d agent.Dialer, c clustering.Cluster, err error) {
	var (
		details agent.ConnectResponse
	)

	conn := newConnect(options...)
	dopts := agent.DefaultDialerOptions(grpc.WithTransportCredentials(creds))
	if cl, err = agent.AddressProxyDialQuorum(config.Address, dopts...); err != nil {
		return cl, d, c, errors.Wrap(err, "proxy dial quorum failed")
	}

	if details, err = cl.Connect(); err != nil {
		return cl, d, c, err
	}

	if c, err = clusterConnect(details, conn.clustering.Options, conn.clustering.Bootstrap); err != nil {
		return cl, d, c, err
	}

	return cl, agent.NewDialer(dopts...), c, nil
}

func clusterConnect(details agent.ConnectResponse, copts []clustering.Option, bopts []clustering.BootstrapOption) (c clustering.Cluster, err error) {
	keyring, err := memberlist.NewKeyring([][]byte{}, details.Secret)
	if err != nil {
		return c, errors.Wrap(err, "failed to create keyring")
	}

	copts = append([]clustering.Option{
		clustering.OptionBindPort(0),
		clustering.OptionLogOutput(ioutil.Discard),
		clustering.OptionKeyring(keyring),
	}, copts...)

	if c, err = clustering.NewOptions(copts...).NewCluster(); err != nil {
		return c, errors.Wrap(err, "failed to join cluster")
	}

	bopts = append([]clustering.BootstrapOption{
		clustering.BootstrapOptionJoinStrategy(clustering.MinimumPeers(1)),
		clustering.BootstrapOptionAllowRetry(clustering.UnlimitedAttempts),
		clustering.BootstrapOptionPeeringStrategies(
			agent.BootstrapPeers(details.Quorum...),
		),
	}, bopts...)

	if err = clustering.Bootstrap(context.Background(), c, bopts...); err != nil {
		return c, errors.Wrap(err, "failed to connect to cluster")
	}

	return c, nil
}
