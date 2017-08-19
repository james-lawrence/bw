package main

import "github.com/alecthomas/kingpin"

func (t *agentCmd) operatingSystemSpecificConfiguration(parent *kingpin.CmdClause) {
	(&directive{agentCmd: t}).configure(parent.Command("directive", "directive based deployment").Default())
	(&dummy{agentCmd: t}).configure(parent.Command("dummy", "dummy deployment"))
}
