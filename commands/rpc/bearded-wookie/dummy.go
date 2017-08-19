package main

import (
	"log"

	"bitbucket.org/jatone/bearded-wookie/agent"
	"bitbucket.org/jatone/bearded-wookie/deployment"
	"github.com/alecthomas/kingpin"
)

type dummy struct {
	*agentCmd
}

func (t *dummy) configure(cmd *kingpin.CmdClause) error {
	cmd.Action(t.attach)
	return nil
}

func (t *dummy) attach(ctx *kingpin.ParseContext) error {
	log.Println("registering dummy deployer")
	defer log.Println("registered dummy deployer")

	agent.RegisterServer(
		t.server,
		agent.NewServer(
			t.listener.Addr(),
			agent.ServerOptionDeployer(deployment.NewDummyCoordinator()),
		),
	)

	return t.agentCmd.Bind(ctx)
}
