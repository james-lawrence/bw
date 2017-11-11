package main

import (
	"log"

	"github.com/james-lawrence/bw/agent"
	"github.com/james-lawrence/bw/agentutil"
	"github.com/james-lawrence/bw/deployment"
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

	return t.agentCmd.bind(
		func(d *agentutil.Dispatcher, p agent.Peer, _ agent.Config) agent.ServerOption {
			return agent.ServerOptionDeployer(deployment.NewDummyCoordinator(p))
		},
	)
}
