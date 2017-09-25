package agent

import (
	"hash"
	"io"
	"net"

	"bitbucket.org/jatone/bearded-wookie"
	"bitbucket.org/jatone/bearded-wookie/clustering"
	"bitbucket.org/jatone/bearded-wookie/deployment/agent"

	"github.com/pkg/errors"

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

// RegisterLeader ...
func RegisterLeader(s *grpc.Server, srv agent.LeaderServer) {
	agent.RegisterLeaderServer(s, srv)
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
	var (
		conn connect
	)

	for _, opt := range options {
		opt(&conn)
	}

	if err = bw.ExpandAndDecodeFile(conn.Path, config); err != nil {
		return creds, client, c, err
	}

	return config.Connect(conn.clustering.Options, conn.clustering.Bootstrap)
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

type operation interface {
	Visit(Client) error
}

type operationFunc func(c Client) error

func (t operationFunc) Visit(c Client) error {
	return t(c)
}

// ClusterOperation ...
type ClusterOperation struct {
	Cluster     cluster
	AgentPort   string
	DialOptions []grpc.DialOption
}

// Perform ...
func (t ClusterOperation) Perform(v operation) (err error) {
	var (
		client Client
	)

	for _, peer := range t.Cluster.Members() {
		if client, err = DialClient(net.JoinHostPort(peer.Addr.String(), t.AgentPort), t.DialOptions...); err != nil {
			return errors.WithStack(err)
		}
		defer client.Close()

		if err = v.Visit(client); err != nil {
			return errors.WithStack(err)
		}
	}

	return nil
}
