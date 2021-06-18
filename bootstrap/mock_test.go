package bootstrap_test

import (
	"github.com/james-lawrence/bw/deployment"
)

type noopDeployer struct {
	err error
}

func (t noopDeployer) Deploy(dctx *deployment.DeployContext) {
	dctx.Done(t.err)
}
