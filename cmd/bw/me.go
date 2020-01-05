package main

import (
	"github.com/alecthomas/kingpin"
	"github.com/james-lawrence/bw"
	"github.com/james-lawrence/bw/notary"
)

// used to configure the user's environment.
type me struct {
	global *global
}

func (t *me) configure(parent *kingpin.CmdClause) {
	parent.Command("init", "initialize the users credentials").Action(t.exec)
}

func (t *me) exec(ctx *kingpin.ParseContext) (err error) {
	if _, err = notary.NewAutoSigner(bw.DisplayName()); err != nil {
		return err
	}

	return nil
}
