package main

import (
	"path/filepath"

	agent "bitbucket.org/jatone/bearded-wookie/agent"
	"bitbucket.org/jatone/bearded-wookie/deployment"
	"bitbucket.org/jatone/bearded-wookie/directives/shell"

	"github.com/alecthomas/kingpin"
)

type directive struct {
	*agentCmd
	root string
}

func (t *directive) configure(cmd *kingpin.CmdClause) error {
	cmd.Flag("package", "file describing a package directive").
		Default(filepath.Join(".bearded-wookie-deployment-empty")).StringVar(&t.root)
	cmd.Action(t.attach)
	return nil
}

func (t *directive) attach(ctx *kingpin.ParseContext) (err error) {
	var (
		sctx shell.Context
	)

	if sctx, err = shell.GenerateContext(); err != nil {
		return err
	}

	deployments := deployment.New(
		agent.NewDirective(
			agent.DirectiveOptionRoot(t.root),
			agent.DirectiveOptionShellContext(sctx),
		),
	)

	agent.RegisterServer(
		t.server,
		agent.NewServer(
			t.listener.Addr(),
			agent.ServerOptionDeployer(deployments),
		),
	)

	return t.agentCmd.Bind(ctx)
}
