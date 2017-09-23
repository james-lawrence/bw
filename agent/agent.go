package agent

import (
	"hash"
	"io"
	"net"

	"github.com/pkg/errors"

	"google.golang.org/grpc"

	"bitbucket.org/jatone/bearded-wookie/clustering"
	"bitbucket.org/jatone/bearded-wookie/deployment/agent"
)

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
	Cluster     clustering.Cluster
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
