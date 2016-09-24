//+build linux

package main

import (
	"log"

	"bitbucket.org/jatone/bearded-wookie/commands/rpc/adapters"
	"bitbucket.org/jatone/bearded-wookie/deployment"
	"gopkg.in/alecthomas/kingpin.v2"
)

type packagekit struct {
	*core
	packageDir string
}

func (t *packagekit) configure(cmd *kingpin.CmdClause) error {
	cmd.Flag("package-set-dir", "file describing the packages to install").ExistingDirVar(&t.packageDir)
	cmd.Action(t.attach)
	return nil
}

func (t *packagekit) attach(*kingpin.ParseContext) error {
	var (
		err   error
		coord deployment.Coordinator
	)

	log.Println("Package dir", t.packageDir)
	options := []deployment.PackagekitOption{
		deployment.PackagekitOptionPackageFilesDirectory(t.packageDir),
	}

	if coord, err = deployment.NewPackagekit(options...); err != nil {
		return err
	}

	deployments := adapters.Deployment{
		Coordinator: coord,
	}

	return t.core.rpc.server.Register(deployments)
}
