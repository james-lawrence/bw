// Package proxy handles being able to run a deploy from one of the members of the cluster.
// allowing for a deploy to be initiated by a local client and then continued even if that client disconnects.
package proxy

import (
	"context"

	"github.com/james-lawrence/bw/agent"
	"github.com/james-lawrence/bw/agent/dialers"
	"google.golang.org/grpc"
)

func check(d dialers.Defaults) func(ctx context.Context, n *agent.Peer) (*agent.Deploy, error) {
	return func(ctx context.Context, n *agent.Peer) (_d *agent.Deploy, err error) {
		var (
			c    *grpc.ClientConn
			info *agent.StatusResponse
		)

		if c, err = dialers.NewDirect(agent.RPCAddress(n)).Dial(d.Defaults()...); err != nil {
			return _d, err
		}

		defer c.Close()

		if info, err = agent.NewConn(c).Info(ctx); err != nil {
			return _d, err
		}

		if len(info.Deployments) > 0 {
			return info.Deployments[0], nil
		}

		return &agent.Deploy{
			Stage: agent.Deploy_Completed,
		}, nil
	}
}

func deploy(dopts *agent.DeployOptions, archive *agent.Archive, d dialers.Defaults) func(ctx context.Context, n *agent.Peer) (*agent.Deploy, error) {
	return func(ctx context.Context, n *agent.Peer) (_d *agent.Deploy, err error) {
		var (
			c *grpc.ClientConn
		)

		if c, err = dialers.NewDirect(agent.RPCAddress(n)).Dial(d.Defaults()...); err != nil {
			return _d, err
		}
		defer c.Close()

		return agent.NewConn(c).Deploy(ctx, dopts, archive)
	}
}
