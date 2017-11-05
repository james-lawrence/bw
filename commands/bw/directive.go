package main

import (
	"os"

	"bitbucket.org/jatone/bearded-wookie/agent"
	"bitbucket.org/jatone/bearded-wookie/agentutil"
	"bitbucket.org/jatone/bearded-wookie/deployment"
	"bitbucket.org/jatone/bearded-wookie/directives/dynplugin"
	"bitbucket.org/jatone/bearded-wookie/directives/shell"
	"bitbucket.org/jatone/bearded-wookie/downloads"

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
		func(d *agentutil.Dispatcher, p agent.Peer, config agent.Config) agent.ServerOption {
			dlreg := downloads.New(
				downloads.OptionProtocols(
					downloads.NewS3Protocol(),
				),
			)

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
