package bootstrap

import (
	"context"

	"github.com/james-lawrence/bw/agent"
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

// Archive - implements the bootstrap service.
func (t Filesystem) Archive(ctx context.Context, req *agent.ArchiveRequest) (resp *agent.ArchiveResponse, err error) {
	if !t.a.Bootstrap.EnableFilesystem {
		return nil, status.Error(codes.FailedPrecondition, "filesystem: disabled")
	}

	return nil, status.Error(codes.NotFound, "filesystem: latest deployment discovery found no deployments")
}
