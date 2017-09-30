package agent

import (
	"hash"
	"io"

	"bitbucket.org/jatone/bearded-wookie"
	"bitbucket.org/jatone/bearded-wookie/clustering"
	"bitbucket.org/jatone/bearded-wookie/deployment/agent"

	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials"
)

// idealized interfaces, for reference while building so end goal isn't lost.
// type leader interface {
// 	Record(...agent.Message) error
// 	Watch(chan<- agent.Message) error
// }
//
// type agent interface {
// 	// Leader - used to retrieve the status of the leader.
// 	Leader() (agent.Status, error)
// 	// Latest - returns the latest deployment.
// 	Latest() (agent.Archive, error)
// 	// Info about the peer.
// 	// idle, canary, deploying, locked, and the list of recent deployments.
// 	Info() (agent.Status, error)
// 	// Deploy trigger a deploy
// 	Deploy(a agent.Archive) error
// 	// Upload
// 	Upload() (agent.Archive, error)
// }

// RegisterServer ...
func RegisterServer(s *grpc.Server, srv agent.AgentServer) {
	agent.RegisterAgentServer(s, srv)
}

// RegisterQuorum ...
func RegisterQuorum(s *grpc.Server, srv agent.QuorumServer) {
	agent.RegisterQuorumServer(s, srv)
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
func ConnectClient(config *ConfigClient, options ...ConnectOption) (creds credentials.TransportCredentials, client Client, c clustering.Cluster, err error) {
	conn := newConnect(options...)

	if err = bw.ExpandAndDecodeFile(conn.Path, config); err != nil {
		return creds, client, c, err
	}

	return config.Connect(
		ConnectOptionClustering(conn.clustering.Options...),
		ConnectOptionBootstrap(conn.clustering.Bootstrap...),
	)
}

// ConnectLeader ...
func ConnectLeader(config *ConfigClient, options ...ConnectOption) (creds credentials.TransportCredentials, client Client, c clustering.Cluster, err error) {
	conn := newConnect(options...)

	if err = bw.ExpandAndDecodeFile(conn.Path, config); err != nil {
		return creds, client, c, err
	}

	return config.ConnectLeader(
		ConnectOptionClustering(conn.clustering.Options...),
		ConnectOptionBootstrap(conn.clustering.Bootstrap...),
	)
}

type cluster interface {
	Local() agent.Peer
	Peers() []agent.Peer
	Quorum() []agent.Peer
	Connect() agent.ConnectInfo
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
	Send(...agent.Message)
}
