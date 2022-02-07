package main

import (
	"github.com/alecthomas/kingpin"
	"github.com/james-lawrence/bw/deployment"
)

type agentDeploymentCoordinator struct {
	*agentCmd
}

func (t *agentDeploymentCoordinator) configure(cmd *kingpin.CmdClause) error {
	cmd.Action(t.attach)
	return nil
}

func (t *agentDeploymentCoordinator) attach(ctx *kingpin.ParseContext) (err error) {
	return t.agentCmd.bind(deployment.Cached{})
}
