package agent

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"strings"

	"github.com/james-lawrence/bw/x/debugx"
	"github.com/james-lawrence/bw/x/errorsx"
	"github.com/james-lawrence/bw/x/iox"
	"github.com/james-lawrence/bw/x/logx"
)

type connector interface {
	Local() Peer
	Connect() ConnectResponse
}

// Coordinator is in charge of coordinating deployments.
type deployer interface {
	// Deploy trigger a deploy
	Deploy(DeployOptions, Archive) (Deploy, error)
	// Cancel current deploy
	Cancel()
	Deployments() ([]Deploy, error)
	Logs([]byte) io.ReadCloser
}

type noopDeployer struct{}

func (t noopDeployer) Deploy(DeployOptions, Archive) (d Deploy, err error) {
	return d, err
}

func (t noopDeployer) Cancel() {}

func (t noopDeployer) Deployments() ([]Deploy, error) {
	return []Deploy{}, nil
}

func (t noopDeployer) Logs(deploymentID []byte) io.ReadCloser {
	return ioutil.NopCloser(strings.NewReader(fmt.Sprintf("INFO: %s", string(deploymentID))))
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
func ServerOptionCluster(c connector) ServerOption {
	return func(s *Server) {
		s.connector = c
	}
}

// ServerOptionShutdown provide a cancellation function that will cause the agent to self terminate.
func ServerOptionShutdown(cf context.CancelFunc) ServerOption {
	return func(s *Server) {
		s.shutdown = cf
	}
}

// NewServer ...
func NewServer(c connector, options ...ServerOption) Server {
	s := Server{
		shutdown: context.CancelFunc(func() {
			log.Println("shutdown isn't implemented")
		}),
		connector: c,
		Deployer:  noopDeployer{},
	}

	for _, opt := range options {
		opt(&s)
	}

	return s
}

// Server ...
type Server struct {
	shutdown  context.CancelFunc
	Deployer  deployer
	connector connector
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

// Cancel ...
func (t Server) Cancel(ctx context.Context, req *CancelRequest) (_ *CancelResponse, err error) {
	debugx.Println("cancel initiated")
	t.Deployer.Cancel()
	return &CancelResponse{}, nil
}

// Info ...
func (t Server) Info(ctx context.Context, _ *StatusRequest) (*StatusResponse, error) {
	var (
		err error
		d   []Deploy
	)

	tmp := t.connector.Local()

	if d, err = t.Deployer.Deployments(); err != nil {
		d = []Deploy{}
		log.Println("failed to read deployments, defaulting to no deployments", err)
	}

	return &StatusResponse{
		Peer:        &tmp,
		Deployments: deployPointers(d...),
	}, nil
}

// Connect ...
func (t Server) Connect(ctx context.Context, _ *ConnectRequest) (_zeror *ConnectResponse, err error) {
	details := t.connector.Connect()
	return &details, nil
}

// Logs retrieve logs for the given deploy.
func (t Server) Logs(req *LogRequest, out Agent_LogsServer) (err error) {
	const KB16 = 16384
	logs := t.Deployer.Logs(req.DeploymentID)

	buf := bytes.NewBuffer(make([]byte, 0, KB16))
	for err == nil {
		err = errorsx.Compact(
			iox.Error(io.CopyN(buf, logs, KB16)),
			out.Send(&LogResponse{Content: buf.Bytes()}),
		)
		buf.Reset()
	}

	return logx.MaybeLog(errorsx.Compact(iox.IgnoreEOF(err), logs.Close()))
}
