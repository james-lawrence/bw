package bootstrap

import (
	"context"
	"log"

	"github.com/james-lawrence/bw/agent"
	"github.com/james-lawrence/bw/agent/dialers"
	"github.com/james-lawrence/bw/agentutil"
	"github.com/pkg/errors"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

// NewLocal consumes a configuration and generates a bootstrap socket
// for the agent.
func NewLocal(p *agent.Peer, d dialer) Local {
	return Local{p: p, d: d}
}

// Local bootstrap service returns the latest deployment of the local agent.
type Local struct {
	agent.UnimplementedBootstrapServer
	p *agent.Peer
	d dialer
}

// Bind the bootstrap service to the provided socket.
func (t Local) Bind(ctx context.Context, socket string, options ...grpc.ServerOption) error {
	return Run(ctx, socket, t, options...)
}

// Archive - implements the bootstrap service.
func (t Local) Archive(ctx context.Context, req *agent.ArchiveRequest) (resp *agent.ArchiveResponse, err error) {
	var (
		latest agent.Deploy
	)

	log.Println("Local.Archive initiated")
	defer log.Println("Local.Archive completed")
	d := dialers.NewDirect(agent.RPCAddress(t.p), t.d.Defaults()...)
	if latest, err = agentutil.LocalLatestDeployment(d); err != nil {
		switch cause := errors.Cause(err); cause {
		case agentutil.ErrNoDeployments:
			return nil, status.Error(codes.NotFound, errors.Wrap(cause, "local: latest deployment discovery found no deployments").Error())
		default:
			return nil, status.Error(codes.Internal, errors.Wrap(cause, "local: failed to determine latest archive to bootstrap").Error())
		}
	}

	return &agent.ArchiveResponse{
		Deploy: &latest,
	}, nil
}
