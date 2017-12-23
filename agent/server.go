package agent

import (
	"context"
	"log"

	"github.com/james-lawrence/bw/x/debugx"
)

// Coordinator is in charge of coordinating deployments.
type deployer interface {
	// Deploy trigger a deploy
	Deploy(*Archive) (Deploy, error)
	Deployments() (Deploy, []*Archive, error)
}

type noopDeployer struct{}

func (t noopDeployer) Deploy(*Archive) (d Deploy, err error) {
	return d, err
}

func (t noopDeployer) Deployments() (Deploy, []*Archive, error) {
	return Deploy{Stage: Deploy_Completed}, []*Archive(nil), nil
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

// ServerOptionShutdown provide a cancellation function that will cause the agent to self terminate.
func ServerOptionShutdown(cf context.CancelFunc) ServerOption {
	return func(s *Server) {
		s.shutdown = cf
	}
}

// NewServer ...
func NewServer(c cluster, options ...ServerOption) Server {
	s := Server{
		shutdown: context.CancelFunc(func() {
			log.Println("shutdown isn't implemented")
		}),
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
	shutdown context.CancelFunc
	Deployer deployer
	cluster  cluster
}

// Shutdown when invoked the agent will self shutdown.
func (t Server) Shutdown(ctx context.Context, req *ShutdownRequest) (*ShutdownResponse, error) {
	t.shutdown()
	return &ShutdownResponse{}, nil
}

// Deploy ...
func (t Server) Deploy(ctx context.Context, archive *Archive) (*ArchiveResult, error) {
	var (
		err error
		d Deploy
	)
	debugx.Println("deploy initiated", archive.Location)
	if d, err = t.Deployer.Deploy(archive); err != nil {
		return nil, err
	}

	return &ArchiveResult{Deploy: &d}, nil
}

// Info ...
func (t Server) Info(ctx context.Context, _ *StatusRequest) (*Status, error) {
	var (
		err error
		d   Deploy
		a   []*Archive
	)

	tmp := t.cluster.Local()

	if d, a, err = t.Deployer.Deployments(); err != nil {
		a = []*Archive{}
		d = Deploy{}
		log.Println("failed to read deployments, defaulting to no deployments", err)
	}

	return &Status{
		Latest:      &d,
		Peer:        &tmp,
		Deployments: a,
	}, nil
}

// Connect ...
func (t Server) Connect(ctx context.Context, _ *ConnectRequest) (_zeror *ConnectInfo, err error) {
	details := t.cluster.Connect()
	return &details, nil
}
