package main

import (
	"log"

	"bitbucket.org/jatone/bearded-wookie/commands/rpc/adapters"
	"bitbucket.org/jatone/bearded-wookie/deployment"
	"github.com/alecthomas/kingpin"
)

type dummy struct {
	*agent
}

func (t *dummy) configure(cmd *kingpin.CmdClause) error {
	cmd.Action(t.attach)
	return nil
}

func (t *dummy) attach(*kingpin.ParseContext) error {
	log.Println("registering dummy deployer")
	defer log.Println("registered dummy deployer")

	deployments := adapters.Deployment{
		Coordinator: deployment.NewDummyCoordinator(),
	}

	return t.agent.server.Register(deployments)
}
