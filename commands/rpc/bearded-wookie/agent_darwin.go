package main

import "github.com/alecthomas/kingpin"

func (t *agent) configure(parent *kingpin.CmdClause) {
	t.cluster.configure(parent)
	parent.Flag("agent-bind", "network interface to listen on").Default("127.0.0.1:2000").TCPVar(&t.network)
	parent.Action(t.Bind)

	(&dummy{agent: t}).configure(parent.Command("dummy", "dummy deployment").Default())
}
