package main

import (
	"github.com/alecthomas/kingpin"

	"github.com/james-lawrence/bw"
)

type workspaceCmd struct {
	global *global
}

func (t *workspaceCmd) configure(parent *kingpin.CmdClause) {
	(&workspaceCreate{global: t.global}).configure(parent.Command("create", "initialize a workspace"))
}

type workspaceCreate struct {
	global          *global
	path            string
	includeExamples bool
}

func (t *workspaceCreate) configure(parent *kingpin.CmdClause) {
	parent.Arg("directory", "path of the workspace directory to create").Default(bw.DefaultDeployspaceDir).StringVar(&t.path)
	parent.Flag("examples", "include examples").Default("false").BoolVar(&t.includeExamples)
	parent.Action(t.generate)
}
