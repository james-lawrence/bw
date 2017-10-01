package agent

import (
	"hash"
	"io"

	"bitbucket.org/jatone/bearded-wookie"
	"bitbucket.org/jatone/bearded-wookie/clustering"

	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials"
)

// Client - client facade interface.
type Client interface {
	Upload(srcbytes uint64, src io.Reader) (Archive, error)
	Deploy(info Archive) error
	Connect() (ConnectInfo, error)
	Info() (Status, error)
	Watch(out chan<- Message) error
	Dispatch(messages ...Message) error
	Close() error
}

// RegisterServer ...
func RegisterServer(s *grpc.Server, srv AgentServer) {
	RegisterAgentServer(s, srv)
}

// RegisterQuorum ...
func RegisterQuorum(s *grpc.Server, srv QuorumServer) {
	RegisterQuorumServer(s, srv)
}

// ConnectOption - options for connecting to the cluster.
type ConnectOption func(*connect)

// ConnectOptionConfigPath path of the configuration file to load.
func ConnectOptionConfigPath(path string) ConnectOption {
	return func(c *connect) {
		c.Path = path
	}
}

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
	Path       string
	clustering struct {
		Options   []clustering.Option
		Bootstrap []clustering.BootstrapOption
		Snapshot  []clustering.SnapshotOption
	}
}

// ConnectClient ...
func ConnectClient(config *ConfigClient, options ...ConnectOption) (creds credentials.TransportCredentials, cl Conn, c clustering.Cluster, err error) {
	conn := newConnect(options...)

	if err = bw.ExpandAndDecodeFile(conn.Path, config); err != nil {
		return creds, cl, c, err
	}

	return config.Connect(
		ConnectOptionClustering(conn.clustering.Options...),
		ConnectOptionBootstrap(conn.clustering.Bootstrap...),
	)
}

// ConnectLeader ...
func ConnectLeader(config *ConfigClient, options ...ConnectOption) (creds credentials.TransportCredentials, cl Conn, c clustering.Cluster, err error) {
	conn := newConnect(options...)

	if err = bw.ExpandAndDecodeFile(conn.Path, config); err != nil {
		return creds, cl, c, err
	}

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
