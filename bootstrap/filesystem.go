package bootstrap

import (
	"context"
	"log"

	"github.com/james-lawrence/bw/agent"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

// NewFilesystem consumes a configuration and generates a bootstrap socket
// for the agent.
func NewFilesystem(a agent.Config, c cluster, d dialer) Filesystem {
	return Filesystem{a: a}
}

// Filesystem bootstrap service will monitor the cluster and write the last
// successful deployment to the filesystem and return that deployment when queried.
// this is useful for storing a backup copy that can be treated as bootstrappable archive.
type Filesystem struct {
	a agent.Config
}

// Bind the bootstrap service to the provided socket.
func (t Filesystem) Bind(ctx context.Context, socket string, options ...grpc.ServerOption) error {
	if !t.a.Bootstrap.EnableFilesystem {
		log.Println("filesystem bootstrap: disabled")
		return nil
	}

	return Run(ctx, socket, t, options...)
}

// Archive - implements the bootstrap service.
func (t Filesystem) Archive(ctx context.Context, req *agent.ArchiveRequest) (resp *agent.ArchiveResponse, err error) {
	return nil, status.Error(codes.NotFound, "filesystem: latest deployment discovery found no deployments")
}
