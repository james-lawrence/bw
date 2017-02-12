package main

import (
	"path/filepath"

	"bitbucket.org/jatone/bearded-wookie/commands/rpc/adapters"
	"bitbucket.org/jatone/bearded-wookie/deployment"
	"bitbucket.org/jatone/bearded-wookie/x/systemx"

	"github.com/alecthomas/kingpin"
	"github.com/pkg/errors"
)

type packagekit struct {
	*agent
	packageDir string
}

func (t *packagekit) configure(cmd *kingpin.CmdClause) error {
	user := systemx.MustUser()

	cmd.Flag("package-set-dir", "file describing the packages to install").
		Default(filepath.Join(user.HomeDir, ".config", "bearded-wookie")).StringVar(&t.packageDir)
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
