package main

import "github.com/alecthomas/kingpin"

func (t *agent) operatingSystemSpecificConfiguration(parent *kingpin.CmdClause) {
	(&packagekit{agent: t}).configure(parent.Command("packagekit", "packagekit deployment").Default())
	(&dummy{agent: t}).configure(parent.Command("dummy", "dummy deployment"))
}
