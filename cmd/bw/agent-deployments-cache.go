package main

import (
	"github.com/alecthomas/kingpin"
	"github.com/james-lawrence/bw/deployment"
)

type agentDeploymentCache struct {
	*agentCmd
}

func (t *agentDeploymentCache) configure(cmd *kingpin.CmdClause) error {
	cmd.Action(t.attach)
	return nil
}

func (t *agentDeploymentCache) attach(ctx *kingpin.ParseContext) (err error) {
	return t.agentCmd.bind(deployment.Cached{})
}
