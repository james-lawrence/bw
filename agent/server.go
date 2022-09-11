package agent

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"log"
	"strings"

	"github.com/james-lawrence/bw/internal/bytesx"
	"github.com/james-lawrence/bw/internal/errorsx"
	"github.com/james-lawrence/bw/internal/iox"
	"github.com/james-lawrence/bw/internal/logx"
	"github.com/pkg/errors"
	"google.golang.org/grpc"
)

type connector interface {
	Local() *Peer
	Connect() ConnectResponse
}

// Coordinator is in charge of coordinating deployments.
type deployer interface {
	// Deploy trigger a deploy
	Deploy(context.Context, *DeployOptions, *Archive) (*Deploy, error)
	// Cancel current deploy
	Cancel()
	Reset() error
	Deployments() ([]*Deploy, error)
	Logs([]byte) io.ReadCloser
}

type noopDeployer struct{}

func (t noopDeployer) Deploy(context.Context, *DeployOptions, *Archive) (d *Deploy, err error) {
	return &Deploy{}, err
}

func (t noopDeployer) Reset() error { return nil }
func (t noopDeployer) Cancel()      {}

func (t noopDeployer) Deployments() ([]*Deploy, error) {
	return []*Deploy{}, nil
}

func (t noopDeployer) Logs(deploymentID []byte) io.ReadCloser {
	return io.NopCloser(strings.NewReader(fmt.Sprintf("INFO: %s", string(deploymentID))))
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

// ServerOptionShutdown provide a cancellation function that will cause the agent to self terminate.
func ServerOptionShutdown(cf context.CancelFunc) ServerOption {
	return func(s *Server) {
		s.shutdown = cf
	}
}

// ServerOptionAuth ...
func ServerOptionAuth(a auth) ServerOption {
	return func(s *Server) {
		s.auth = a
	}
}

// NewServer ...
func NewServer(c connector, options ...ServerOption) Server {
	s := Server{
		auth: noauth{},
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
	UnimplementedAgentServer
	auth
	shutdown  context.CancelFunc
	Deployer  deployer
	connector connector
}

// Bind to a grpc server.
func (t Server) Bind(srv *grpc.Server) Server {
	RegisterAgentServer(srv, t)
	return t
}

// Shutdown when invoked the agent will self shutdown.
func (t Server) Shutdown(ctx context.Context, req *ShutdownRequest) (*ShutdownResponse, error) {
	if err := t.auth.Deploy(ctx); err != nil {
		return nil, err
	}

	if err := logx.MaybeLog(errors.Wrap(t.Deployer.Reset(), "failed to reset")); err != nil {
		return nil, err
	}

	t.shutdown()
	return &ShutdownResponse{}, nil
}

// Deploy ...
func (t Server) Deploy(ctx context.Context, dreq *DeployRequest) (*DeployResponse, error) {
	var (
		err error
		d   *Deploy
	)

	if err := t.auth.Deploy(ctx); err != nil {
		return nil, err
	}

	if d, err = t.Deployer.Deploy(context.Background(), dreq.Options, dreq.Archive); err != nil {
		return nil, err
	}

	return &DeployResponse{Deploy: d}, nil
}

// Cancel ...
func (t Server) Cancel(ctx context.Context, req *CancelRequest) (_ *CancelResponse, err error) {
	if err := t.auth.Deploy(ctx); err != nil {
		return nil, err
	}

	t.Deployer.Cancel()
	return &CancelResponse{}, nil
}

// Info ...
func (t Server) Info(ctx context.Context, _ *StatusRequest) (_ *StatusResponse, err error) {
	var (
		d []*Deploy
	)

	if err := t.auth.Deploy(ctx); err != nil {
		return nil, err
	}

	tmp := t.connector.Local()

	if d, err = t.Deployer.Deployments(); err != nil {
		d = []*Deploy{}
		log.Println("failed to read deployments, defaulting to no deployments", err)
	}

	return &StatusResponse{
		Peer:        tmp,
		Deployments: d,
	}, nil
}

// Connect ...
func (t Server) Connect(ctx context.Context, _ *ConnectRequest) (_zeror *ConnectResponse, err error) {
	if err := t.auth.Deploy(ctx); err != nil {
		return nil, err
	}

	details := t.connector.Connect()
	return &details, nil
}

// Logs retrieve logs for the given deploy.
func (t Server) Logs(req *LogRequest, out Agent_LogsServer) (err error) {
	if err := t.auth.Deploy(out.Context()); err != nil {
		return err
	}

	const KB16 = 16 * bytesx.KiB
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
