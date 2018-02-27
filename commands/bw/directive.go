package main

import (
	"github.com/james-lawrence/bw/agent"
	"github.com/james-lawrence/bw/deployment"
	"github.com/james-lawrence/bw/directives/shell"
	"github.com/james-lawrence/bw/storage"

	"github.com/alecthomas/kingpin"
)

type agentContext struct {
	Config           agent.Config
	Dispatcher       agent.Dispatcher
	completedDeploys chan deployment.DeployResult
}

type directive struct {
	*agentCmd
}

func (t *directive) configure(cmd *kingpin.CmdClause) error {
	cmd.Action(t.attach)
	return nil
}

func (t *directive) attach(ctx *kingpin.ParseContext) (err error) {
	var (
		sctx shell.Context
	)

	if sctx, err = shell.DefaultContext(); err != nil {
		return err
	}

	return t.agentCmd.bind(
		func(actx agentContext, dl storage.DownloadProtocol) deployment.Coordinator {
			var (
				dlreg = storage.New(storage.OptionDefaultProtocols(actx.Config.Root, dl))
			)

			deploy := deployment.NewDirective(
				deployment.DirectiveOptionShellContext(sctx),
			)

			deployments := deployment.New(
				actx.Config.Peer(),
				deploy,
				deployment.CoordinatorOptionDispatcher(actx.Dispatcher),
				deployment.CoordinatorOptionRoot(actx.Config.Root),
				deployment.CoordinatorOptionKeepN(actx.Config.KeepN),
				deployment.CoordinatorOptionDeployResults(actx.completedDeploys),
				deployment.CoordinatorOptionStorage(dlreg),
			)

			return deployments
		},
	)
}
