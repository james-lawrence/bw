package agent

import (
	"context"
	"hash"
	"io"
	"time"

	"github.com/james-lawrence/bw/clustering"
	"github.com/james-lawrence/bw/x/logx"
	"github.com/pkg/errors"
)

// Dispatcher - interface for dispatching messages.
type Dispatcher interface {
	Dispatch(...Message) error
}

// Client - client facade interface.
type Client interface {
	Shutdown() error
	Upload(srcbytes uint64, src io.Reader) (Archive, error)
	RemoteDeploy(dopts DeployOptions, a Archive, peers ...Peer) error
	Deploy(DeployOptions, Archive) (Deploy, error)
	Connect() (ConnectResponse, error)
	Info() (StatusResponse, error)
	Watch(out chan<- Message) error
	Dispatch(messages ...Message) error
	Close() error
}

// ConnectOption - options for connecting to the cluster.
type ConnectOption func(*connect)

// ConnectOptionClustering set clustering options for connect.
func ConnectOptionClustering(options ...clustering.Option) ConnectOption {
	return func(c *connect) {
		c.clustering.Options = options
	}
}

// ConnectOptionBootstrap set clustering options for bootstrap.
func ConnectOptionBootstrap(options ...clustering.BootstrapOption) ConnectOption {
	return func(c *connect) {
		c.clustering.Bootstrap = options
	}
}

func newConnect(options ...ConnectOption) connect {
	var (
		conn connect
	)

	for _, opt := range options {
		opt(&conn)
	}

	return conn
}

type connect struct {
	clustering struct {
		Options   []clustering.Option
		Bootstrap []clustering.BootstrapOption
		Snapshot  []clustering.SnapshotOption
	}
}

// ConnectClientUntilSuccess continuously tries to make a connection until successful.
func ConnectClientUntilSuccess(
	ctx context.Context,
	config ConfigClient, onRetry func(error), options ...ConnectOption,
) (client Client, c clustering.Cluster, err error) {
	for i := 0; true; i++ {
		if client, c, err = Connect(config, options...); err == nil {
			break
		}

		// when an error occurs, cleanup any resources.
		logx.MaybeLog(errors.WithMessage(c.Shutdown(), "failed to cleanup cluster"))

		select {
		case <-ctx.Done():
			return client, c, ctx.Err()
		default:
		}

		onRetry(err)
		time.Sleep(250 * time.Millisecond)
	}

	return client, c, err
}

// Connect returning just a single client to the caller.
func Connect(config ConfigClient, options ...ConnectOption) (cl Client, c clustering.Cluster, err error) {
	conn := newConnect(options...)
	cl, _, c, err = config.Connect(
		ConnectOptionClustering(conn.clustering.Options...),
		ConnectOptionBootstrap(conn.clustering.Bootstrap...),
	)

	return cl, c, err
}

type cluster interface {
	Local() Peer
	Peers() []Peer
	Quorum() []Peer
	Connect() ConnectResponse
}

// downloader ...
type downloader interface {
	Download() io.ReadCloser
}

// Uploader ...
type Uploader interface {
	Upload(io.Reader) (hash.Hash, error)
	Info() (hash.Hash, string, error)
}
