package deployment

import "github.com/james-lawrence/bw/agent"

// NewNopCoordinator Builds a coordinator that uses a fake deployer.
func NewNopCoordinator(result error, p *agent.Peer, options ...CoordinatorOption) Coordinator {
	return New(p, nop{result: result}, options...)
}

type nop struct {
	result error
}

func (t nop) Deploy(dctx *DeployContext) {
	dctx.Done(t.result)
}
