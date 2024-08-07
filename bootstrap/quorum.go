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

// NewQuorum consumes a configuration and generates a bootstrap socket
// for the agent. This bootstrap socket provides an archive based on what the
// cluster's raft porotocol considers the latest deployment.
func NewQuorum(c cluster, d dialer) Quorum {
	return Quorum{c: c, d: d}
}

// Quorum implements the cluster bootstrap service.
type Quorum struct {
	agent.UnimplementedBootstrapServer
	c cluster
	d dialer
}

// Bind the bootstrap service to the provided socket.
func (t Quorum) Bind(ctx context.Context, socket string, options ...grpc.ServerOption) error {
	return Run(ctx, socket, t, options...)
}

// Archive - implements the bootstrap service.
func (t Quorum) Archive(ctx context.Context, req *agent.ArchiveRequest) (resp *agent.ArchiveResponse, err error) {
	var (
		latest *agent.Deploy
	)

	if latest, err = agentutil.QuorumLatestDeployment(t.c, t.d.Defaults()); err != nil {
		switch cause := errors.Cause(err); cause {
		case agentutil.ErrActiveDeployment:
			return &agent.ArchiveResponse{
				Info:   agent.ArchiveResponse_ActiveDeploy,
				Deploy: latest,
			}, nil
		case agentutil.ErrNoDeployments:
			return nil, status.Error(codes.NotFound, errors.Wrap(cause, "quorum: no deployments").Error())
		default:
			return nil, status.Error(codes.Unavailable, errors.Wrap(cause, "quorum: failed to determine latest archive to bootstrap").Error())
		}
	}

	return &agent.ArchiveResponse{
		Deploy: latest,
	}, nil
}
