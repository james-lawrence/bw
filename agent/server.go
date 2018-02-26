package agent

import (
	"context"
	"log"

	"github.com/james-lawrence/bw/x/debugx"
)

// Coordinator is in charge of coordinating deployments.
type deployer interface {
	// Deploy trigger a deploy
	Deploy(DeployOptions, Archive) (Deploy, error)
	Deployments() ([]Deploy, error)
}

type noopDeployer struct{}

func (t noopDeployer) Deploy(DeployOptions, Archive) (d Deploy, err error) {
	return d, err
}

func (t noopDeployer) Deployments() ([]Deploy, error) {
	return []Deploy{}, nil
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
func (t Server) Deploy(ctx context.Context, dreq *DeployRequest) (*DeployResponse, error) {
	var (
		err error
		d   Deploy
	)

	debugx.Println("deploy initiated", dreq.Archive.Location)
	if d, err = t.Deployer.Deploy(*dreq.Options, *dreq.Archive); err != nil {
		return nil, err
	}

	return &DeployResponse{Deploy: &d}, nil
}

// Info ...
func (t Server) Info(ctx context.Context, _ *StatusRequest) (*StatusResponse, error) {
	var (
		err error
		d   []Deploy
	)

	tmp := t.cluster.Local()

	if d, err = t.Deployer.Deployments(); err != nil {
		d = []Deploy{}
		log.Println("failed to read deployments, defaulting to no deployments", err)
	}

	// these fields are deprecated and can just use the first deployment (if any)
	// to determine latest.
	ddeprecated := deploysFirstOrDefault(Deploy{Stage: Deploy_Completed}, d...)
	adeprecated := deployArchives(d...)
	return &StatusResponse{
		Peer:        &tmp,
		Latest:      &ddeprecated,
		Archives:    adeprecated,
		Deployments: deployPointers(d...),
	}, nil
}

// Connect ...
func (t Server) Connect(ctx context.Context, _ *ConnectRequest) (_zeror *ConnectResponse, err error) {
	details := t.cluster.Connect()
	return &details, nil
}
