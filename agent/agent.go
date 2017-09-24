package agent

import (
	"hash"
	"io"
	"net"

	"bitbucket.org/jatone/bearded-wookie/deployment/agent"

	"github.com/pkg/errors"

	"google.golang.org/grpc"
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
