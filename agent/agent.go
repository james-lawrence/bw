package agent

import (
	"context"
	"hash"
	"io"
	"log"
	"time"

	"github.com/james-lawrence/bw/backoff"
	"github.com/james-lawrence/bw/clustering"

	"google.golang.org/grpc/credentials"
)

// Client - client facade interface.
type Client interface {
	Shutdown() error
	Upload(srcbytes uint64, src io.Reader) (Archive, error)
	RemoteDeploy(concurrency int64, archive Archive, peers ...Peer) error
	Deploy(info Archive) (Deploy, error)
	Connect() (ConnectInfo, error)
	Info() (Status, error)
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
func ConnectClientUntilSuccess(ctx context.Context, delay backoff.Strategy, config ConfigClient, options ...ConnectOption) (creds credentials.TransportCredentials, cl Conn, c clustering.Cluster, err error) {
	for i := 0; true; i++ {
		if creds, cl, c, err = ConnectClient(config, options...); err == nil {
			break
		}

		// cleanup any resources that where created.
		cl.Close()
		c.Shutdown()

		select {
		case <-ctx.Done():
			return creds, cl, c, ctx.Err()
		default:
		}

		log.Println("failed to connect to cluster", err)
		time.Sleep(delay.Backoff(i))
	}

	return creds, cl, c, err
}

// ConnectClient ...
func ConnectClient(config ConfigClient, options ...ConnectOption) (creds credentials.TransportCredentials, cl Conn, c clustering.Cluster, err error) {
	conn := newConnect(options...)

	return config.Connect(
		ConnectOptionClustering(conn.clustering.Options...),
		ConnectOptionBootstrap(conn.clustering.Bootstrap...),
	)
}

// ConnectLeader ...
func ConnectLeader(config ConfigClient, options ...ConnectOption) (creds credentials.TransportCredentials, cl Conn, c clustering.Cluster, err error) {
	conn := newConnect(options...)

	return config.ConnectLeader(
		ConnectOptionClustering(conn.clustering.Options...),
		ConnectOptionBootstrap(conn.clustering.Bootstrap...),
	)
}

type cluster interface {
	Local() Peer
	Peers() []Peer
	Quorum() []Peer
	Connect() ConnectInfo
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

// Eventer ...
type Eventer interface {
	Send(...Message)
}
