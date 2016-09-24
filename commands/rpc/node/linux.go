//+build linux

package main

import "gopkg.in/alecthomas/kingpin.v2"

func configure(system *core, app *kingpin.Application) error {
	var (
		cmd *kingpin.CmdClause
	)

	cmd = app.Command("dummy", "dummy deployment")
	dummy{core: system}.configure(cmd)
	cmd = app.Command("packagekit", "packagekit based deployment")
	(&packagekit{core: system}).configure(cmd)

	return nil
}
