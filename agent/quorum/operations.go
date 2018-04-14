package quorum

import (
	"github.com/james-lawrence/bw/agent"
)

func check(d agent.Dialer) func(n agent.Peer) (agent.Deploy, error) {
	return func(n agent.Peer) (_d agent.Deploy, err error) {
		var (
			c    agent.Client
			info agent.StatusResponse
		)

		if c, err = d.Dial(n); err != nil {
			return _d, err
		}

		defer c.Close()

		if info, err = c.Info(); err != nil {
			return _d, err
		}

		if len(info.Deployments) > 0 {
			return *info.Deployments[0], nil
		}

		return agent.Deploy{
			Stage: agent.Deploy_Completed,
		}, nil
	}
}

func deploy(dopts agent.DeployOptions, archive agent.Archive, d agent.Dialer) func(n agent.Peer) (agent.Deploy, error) {
	return func(n agent.Peer) (_d agent.Deploy, err error) {
		var (
			c agent.Client
		)

		if c, err = d.Dial(n); err != nil {
			return _d, err
		}
		defer c.Close()

		return c.Deploy(dopts, archive)
	}
}
