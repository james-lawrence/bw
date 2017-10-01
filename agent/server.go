package agent

import (
	"net"

	"google.golang.org/grpc/credentials"

	"golang.org/x/net/context"
)

// Coordinator is in charge of coordinating deployments.
type deployer interface {
	// Deploy trigger a deploy
	Deploy(*Archive) error
}

type noopDeployer struct{}

func (t noopDeployer) Deploy(*Archive) error {
	return nil
}

// ServerOption ...
type ServerOption func(*Server)

// ComposeServerOptions turns a set of server options into a single server option.
func ComposeServerOptions(options ...ServerOption) ServerOption {
	return func(s *Server) {
		for _, opt := range options {
			opt(s)
		}
	}
}

// ServerOptionDeployer ...
func ServerOptionDeployer(d deployer) ServerOption {
	return func(s *Server) {
		s.Deployer = d
	}
}

// ServerOptionCluster ...
func ServerOptionCluster(c cluster) ServerOption {
	return func(s *Server) {
		s.cluster = c
	}
}

// NewServer ...
func NewServer(c cluster, address net.Addr, creds credentials.TransportCredentials, options ...ServerOption) Server {
	s := Server{
		creds:    creds,
		cluster:  c,
		Deployer: noopDeployer{},
	}

	for _, opt := range options {
		opt(&s)
	}

	return s
}

// Server ...
type Server struct {
	creds    credentials.TransportCredentials
	Deployer deployer
	cluster  cluster
}

// Deploy ...
func (t Server) Deploy(ctx context.Context, archive *Archive) (*ArchiveResult, error) {
	if err := t.Deployer.Deploy(archive); err != nil {
		return nil, err
	}

	return &ArchiveResult{}, nil
}

// Info ...
func (t Server) Info(ctx context.Context, _ *StatusRequest) (*Status, error) {
	tmp := t.cluster.Local()
	return &Status{
		Peer: &tmp,
	}, nil
}

// Connect ...
func (t Server) Connect(ctx context.Context, _ *ConnectRequest) (_zeror *ConnectInfo, err error) {
	details := t.cluster.Connect()
	return &details, nil
}
