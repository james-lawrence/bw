package main

import (
	"github.com/alecthomas/kingpin"
	"github.com/james-lawrence/bw/deployment"
	"github.com/james-lawrence/bw/directives/shell"
)

type agentDeploymentRuntime struct {
	*agentCmd
}

func (t *agentDeploymentRuntime) configure(cmd *kingpin.CmdClause) error {
	cmd.Action(t.attach)
	return nil
}

func (t *agentDeploymentRuntime) attach(ctx *kingpin.ParseContext) (err error) {
	var (
		sctx shell.Context
	)

	if sctx, err = shell.DefaultContext(); err != nil {
		return err
	}

	return t.agentCmd.bind(deployment.NewDirective(
		deployment.DirectiveOptionShellContext(sctx),
	))
}
