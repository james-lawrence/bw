package deployment

import (
	"github.com/james-lawrence/bw/agent"
	"github.com/james-lawrence/bw/internal/errorsx"
)

// NewNopCoordinator Builds a coordinator that uses a fake deployer.
func NewNopCoordinator(result error, p *agent.Peer, options ...CoordinatorOption) Coordinator {
	return New(p, nop{result: result}, options...)
}

type nop struct {
	result error
}

func (t nop) Deploy(dctx *DeployContext) {
	errorsx.Log(dctx.Done(t.result))
}
