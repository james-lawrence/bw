package bootstrap

import (
	"context"

	"github.com/james-lawrence/bw/agent"
	"github.com/james-lawrence/bw/agentutil"
	"github.com/pkg/errors"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

// NewCluster consumes a configuration and generates a bootstrap socket
// for the agent.
func NewCluster(c cluster, d dialer) Cluster {
	return Cluster{c: c, d: d}
}

// Cluster implements the cluster bootstrap service by polling all known agents
// within the cluster for that latest successful deploy and returning the deploy
// that exceeds 50% of the cluster's agents.
type Cluster struct {
	agent.UnimplementedBootstrapServer
	c cluster
	d dialer
}

// Bind the bootstrap service to the provided socket.
func (t Cluster) Bind(ctx context.Context, socket string, options ...grpc.ServerOption) error {
	return Run(ctx, socket, t, options...)
}

// Archive - implements the bootstrap service.
func (t Cluster) Archive(ctx context.Context, req *agent.ArchiveRequest) (resp *agent.ArchiveResponse, err error) {
	var (
		latest agent.Deploy
	)

	if latest, err = agentutil.DetermineLatestDeployment(t.c, t.d.Defaults()); err != nil {
		switch cause := errors.Cause(err); cause {
		case agentutil.ErrNoDeployments:
			return nil, status.Error(codes.NotFound, errors.Wrap(cause, "cluster: latest deployment discovery found no deployments").Error())
		default:
			return nil, status.Error(codes.Internal, errors.Wrap(cause, "cluster: failed to determine latest archive to bootstrap").Error())
		}
	}

	return &agent.ArchiveResponse{
		Deploy: &latest,
	}, nil
}
