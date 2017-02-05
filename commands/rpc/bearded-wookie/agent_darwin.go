package main

import "github.com/alecthomas/kingpin"

func (t *agent) operatingSystemSpecificConfiguration(parent *kingpin.CmdClause) {
	(&dummy{agent: t}).configure(parent.Command("dummy", "dummy deployment").Default())
}
