package deployment

import "github.com/james-lawrence/bw/agent"

// NewSleepyCoordinator Builds a coordinator that uses a fake deployer.
func NewNopCoordinator(p agent.Peer) Coordinator {
	return New(p, nop{})
}

type nop struct{}

func (t nop) Deploy(dctx DeployContext) {
}
