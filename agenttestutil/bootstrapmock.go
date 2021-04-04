package agenttestutil

import (
	"context"

	"github.com/james-lawrence/bw/agent"
	"google.golang.org/grpc"
)

// Mock bootstrap service
type Mock struct {
	agent.UnimplementedBootstrapServer
	Fail    error
	Current *agent.Deploy
	Info    agent.ArchiveResponse_Info
}

func (t *Mock) Bind(srv *grpc.Server) {
	agent.RegisterBootstrapServer(srv, t)
}

// Archive - implements the bootstrap service.
func (t *Mock) Archive(ctx context.Context, req *agent.ArchiveRequest) (resp *agent.ArchiveResponse, err error) {
	return &agent.ArchiveResponse{
		Info:   t.Info,
		Deploy: t.Current,
	}, t.Fail
}
