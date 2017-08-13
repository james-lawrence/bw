package main

import (
	"path/filepath"

	"bitbucket.org/jatone/bearded-wookie/commands/rpc/adapters"
	"bitbucket.org/jatone/bearded-wookie/deployment"
	"bitbucket.org/jatone/bearded-wookie/directives/shell"

	"github.com/alecthomas/kingpin"
	"github.com/pkg/errors"
)

type directive struct {
	*agent
	packageDirective string
}

func (t *directive) configure(cmd *kingpin.CmdClause) error {
	cmd.Flag("package", "file describing a package directive").
		Default(filepath.Join(".bearded-wookie-deployment")).StringVar(&t.packageDirective)
	cmd.Action(t.attach)
	return nil
}

func (t *directive) attach(*kingpin.ParseContext) (err error) {
	var (
		sctx shell.Context
	)

	if sctx, err = shell.GenerateContext(); err != nil {
		return err
	}

	deployments := adapters.Deployment{
		Coordinator: deployment.New(deployment.NewDirective(
			deployment.DirectiveOptionBaseDirectory(t.packageDirective),
			deployment.DirectiveOptionShellContext(sctx),
		)),
	}

	if err = t.agent.server.Register(deployments); err != nil {
		return errors.Wrap(err, "failed to register agent with rpc server")
	}

	return nil
}
