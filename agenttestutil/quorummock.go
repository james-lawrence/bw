package agenttestutil

import (
	"context"

	"github.com/james-lawrence/bw/agent"
	"google.golang.org/grpc"
)

type FakeQuorum struct {
	agent.UnimplementedQuorumServer
	agent.DeployCommandResult
	agent.DispatchResponse
	agent.InfoResponse
	ErrResult error
}

func (t *FakeQuorum) Bind(srv *grpc.Server) {
	agent.RegisterQuorumServer(srv, t)
}

func (t *FakeQuorum) Deploy(ctx context.Context, req *agent.DeployCommandRequest) (*agent.DeployCommandResult, error) {
	return &t.DeployCommandResult, t.ErrResult
}

func (t *FakeQuorum) Dispatch(_ context.Context, req *agent.DispatchRequest) (*agent.DispatchResponse, error) {
	return &t.DispatchResponse, t.ErrResult
}

func (t *FakeQuorum) Info(ctx context.Context, req *agent.InfoRequest) (*agent.InfoResponse, error) {
	return &t.InfoResponse, t.ErrResult
}

func (t *FakeQuorum) Upload(s agent.Quorum_UploadServer) error {
	return t.ErrResult
}

func (t *FakeQuorum) Watch(req *agent.WatchRequest, s agent.Quorum_WatchServer) error {
	return t.ErrResult
}
