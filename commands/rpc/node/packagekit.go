//+build linux

package main

import (
	"bitbucket.org/jatone/bearded-wookie/commands/rpc/adapters"
	"bitbucket.org/jatone/bearded-wookie/deployment"
	"gopkg.in/alecthomas/kingpin.v2"
)

type packagekit struct {
	*core
	packageFiles []string
}

func (t *packagekit) configure(cmd *kingpin.CmdClause) error {
	cmd.Flag("package-set", "file describing the packages to install").ExistingFilesVar(&t.packageFiles)
	cmd.Action(t.attach)
	return nil
}

func (t *packagekit) attach(*kingpin.ParseContext) error {
	deployments := adapters.Deployment{
		Coordinator: deployment.NewPackagekit(),
	}

	return t.core.rpc.server.Register(deployments)
}
