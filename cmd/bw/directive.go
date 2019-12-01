package main

import (
	"github.com/james-lawrence/bw/agent"
	"github.com/james-lawrence/bw/deployment"

	"github.com/alecthomas/kingpin"
)

type agentContext struct {
	Config           agent.Config
	Dispatcher       agent.Dispatcher
	completedDeploys chan deployment.DeployResult
}

type directive struct {
	*agentCmd
}

func (t *directive) configure(cmd *kingpin.CmdClause) error {
	cmd.Action(t.attach)
	return nil
}

func (t *directive) attach(ctx *kingpin.ParseContext) (err error) {
	return t.agentCmd.bind()
}
