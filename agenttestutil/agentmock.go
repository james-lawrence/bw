package agenttestutil

import (
	"context"

	"github.com/james-lawrence/bw/agent"
	"google.golang.org/grpc"
)

type FakeAgent struct {
	agent.UnimplementedAgentServer
	ErrResult       error
	Archive         agent.Archive
	DeployResponse  agent.Deploy
	ConnectResponse agent.ConnectResponse
	StatusResponse  agent.StatusResponse
}

func (t *FakeAgent) Bind(srv *grpc.Server) {
	agent.RegisterAgentServer(srv, t)
}

func (t *FakeAgent) Shutdown(ctx context.Context, req *agent.ShutdownRequest) (*agent.ShutdownResponse, error) {
	return nil, t.ErrResult
}

func (t *FakeAgent) Cancel(ctx context.Context, req *agent.CancelRequest) (*agent.CancelResponse, error) {
	return nil, t.ErrResult
}

func (t *FakeAgent) NodeCancel() error {
	return t.ErrResult
}

func (t *FakeAgent) Logs(req *agent.LogRequest, s agent.Agent_LogsServer) error {
	return t.ErrResult
}

func (t *FakeAgent) Deploy(ctx context.Context, req *agent.DeployRequest) (*agent.DeployResponse, error) {
	return &agent.DeployResponse{
		Deploy: &t.DeployResponse,
	}, t.ErrResult
}

func (t *FakeAgent) Connect(ctx context.Context, req *agent.ConnectRequest) (*agent.ConnectResponse, error) {
	return &t.ConnectResponse, t.ErrResult
}

func (t *FakeAgent) Info(ctx context.Context, req *agent.StatusRequest) (*agent.StatusResponse, error) {
	return &t.StatusResponse, t.ErrResult
}
