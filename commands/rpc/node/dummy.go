package main

import (
	"bitbucket.org/jatone/bearded-wookie/commands/rpc/adapters"
	"bitbucket.org/jatone/bearded-wookie/deployment"
	"gopkg.in/alecthomas/kingpin.v2"
)

type dummy struct {
	*core
}

func (t dummy) configure(cmd *kingpin.CmdClause) error {
	cmd.Action(t.attach)
	return nil
}

func (t dummy) attach(*kingpin.ParseContext) error {
	deployments := adapters.Deployment{
		Coordinator: deployment.NewDummyCoordinator(),
	}

	return t.core.rpc.server.Register(deployments)
}
