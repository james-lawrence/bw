// Package proxy handles being able to run a deploy from one of the members of the cluster.
// allowing for a deploy to be initiated by a local client and then continued even if that client disconnects.
package proxy

import (
	"github.com/james-lawrence/bw/agent"
	"google.golang.org/grpc"
)

func check(options ...grpc.DialOption) func(n agent.Peer) (agent.Deploy, error) {
	return func(n agent.Peer) (_d agent.Deploy, err error) {
		var (
			c    agent.Client
			info agent.StatusResponse
		)

		if c, err = agent.Dial(agent.RPCAddress(n), options...); err != nil {
			return _d, err
		}

		defer c.Close()

		if info, err = c.Info(); err != nil {
			return _d, err
		}

		// TODO DEPRECATED: Remove latest in next version. use deployments going forward.
		if info.Latest != nil {
			return *info.Latest, nil
		}

		if len(info.Deployments) > 0 {
			return *info.Deployments[0], nil
		}

		return agent.Deploy{
			Stage: agent.Deploy_Completed,
		}, nil
	}
}

func deploy(dopts agent.DeployOptions, archive agent.Archive, options ...grpc.DialOption) func(n agent.Peer) (agent.Deploy, error) {
	return func(n agent.Peer) (_d agent.Deploy, err error) {
		var (
			c agent.Client
		)

		if c, err = agent.Dial(agent.RPCAddress(n), options...); err != nil {
			return _d, err
		}
		defer c.Close()

		return c.Deploy(dopts, archive)
	}
}
