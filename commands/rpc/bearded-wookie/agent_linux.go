package main

import "github.com/alecthomas/kingpin"

func (t *agent) operatingSystemSpecificConfiguration(parent *kingpin.CmdClause) {
	(&directive{agent: t}).configure(parent.Command("directive", "directive based deployment").Default())
	(&dummy{agent: t}).configure(parent.Command("dummy", "dummy deployment"))
}
