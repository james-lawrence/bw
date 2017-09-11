package main

import (
	agent "bitbucket.org/jatone/bearded-wookie/agent"
	"bitbucket.org/jatone/bearded-wookie/deployment"
	"bitbucket.org/jatone/bearded-wookie/directives/shell"

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
		sctx shell.Context
	)

	if sctx, err = shell.DefaultContext(); err != nil {
		return err
	}

	return t.agentCmd.bind(
		t.listener.Addr(),
		func(config agent.Config) agent.ServerOption {
			deployments := deployment.New(
				agent.NewDirective(
					agent.DirectiveOptionShellContext(sctx),
				),
				deployment.CoordinatorOptionRoot(config.Root),
				deployment.CoordinatorOptionKeepN(config.KeepN),
			)
			return agent.ServerOptionDeployer(deployments)
		},
	)
}
