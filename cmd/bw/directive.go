package main

import (
	"github.com/alecthomas/kingpin"
)

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
