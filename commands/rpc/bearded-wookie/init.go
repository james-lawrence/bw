package main

import (
	"github.com/alecthomas/kingpin"
)

type initCmd struct {
	*global
}

func (t *initCmd) configure(parent *kingpin.CmdClause) {
	// cmd := parent.Command("all", "deploy to all nodes within the cluster").Default().Action(t.Deploy)
}

func (t *initCmd) Deploy(ctx *kingpin.ParseContext) (err error) {

	// complete.
	t.shutdown()

	return err
}
