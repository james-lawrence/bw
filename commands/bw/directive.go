package main

import (
	"os"

	"github.com/james-lawrence/bw/agent"
	"github.com/james-lawrence/bw/agentutil"
	"github.com/james-lawrence/bw/deployment"
	"github.com/james-lawrence/bw/directives/dynplugin"
	"github.com/james-lawrence/bw/directives/shell"
	"github.com/james-lawrence/bw/storage"

	"github.com/alecthomas/kingpin"
)

type directive struct {
	*agentCmd
}

func (t *directive) configure(cmd *kingpin.CmdClause) error {
	cmd.Action(t.attach)
	return nil
}

func (t *directive) attach(ctx *kingpin.ParseContext) (err error) {
	var (
		sctx    shell.Context
		plugins []dynplugin.Directive
	)

	if sctx, err = shell.DefaultContext(); err != nil {
		return err
	}

	if plugins, err = dynplugin.Load("./plugins"); !os.IsNotExist(err) && err != nil {
		return err
	}

	return t.agentCmd.bind(
		func(d *agentutil.Dispatcher, p agent.Peer, config agent.Config, dl storage.DownloadProtocol) agent.ServerOption {
			var (
				dlreg storage.Registry
			)

			if dl == nil {
				dlreg = storage.New(storage.OptionDefaultProtocols(config.Root))
			} else {
				dlreg = storage.New(storage.OptionDefaultProtocols(config.Root, dl))
			}


			deployments := deployment.New(
				p,
				deployment.NewDirective(
					deployment.DirectiveOptionShellContext(sctx),
					deployment.DirectiveOptionPlugins(plugins...),
					deployment.DirectiveOptionDownloadRegistry(dlreg),
				),
				deployment.CoordinatorOptionDispatcher(d),
				deployment.CoordinatorOptionRoot(config.Root),
				deployment.CoordinatorOptionKeepN(config.KeepN),
			)
			return agent.ServerOptionDeployer(deployments)
		},
	)
}
