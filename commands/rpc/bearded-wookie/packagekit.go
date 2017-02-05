package main

import (
	"bitbucket.org/jatone/bearded-wookie/commands/rpc/adapters"
	"bitbucket.org/jatone/bearded-wookie/deployment"

	"github.com/pkg/errors"
	"github.com/alecthomas/kingpin"
)

type packagekit struct {
	*agent
	packageDir string
}

func (t *packagekit) configure(cmd *kingpin.CmdClause) error {
	cmd.Flag("package-set-dir", "file describing the packages to install").ExistingDirVar(&t.packageDir)
	cmd.Action(t.attach)
	return nil
}

func (t *packagekit) attach(*kingpin.ParseContext) error {
	var (
		err error
	)
	deployments := adapters.Deployment{
		Coordinator: deployment.NewPackagekit(
			deployment.PackagekitOptionPackageFilesDirectory(t.packageDir),
		),
	}

	if err = t.agent.server.Register(deployments); err != nil {
		return errors.Wrap(err, "failed to register packagekit with rpc server")
	}

	return nil
}
