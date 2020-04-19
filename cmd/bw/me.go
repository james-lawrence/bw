package main

import (
	"log"

	"github.com/alecthomas/kingpin"
	"github.com/james-lawrence/bw"
	"github.com/james-lawrence/bw/notary"
)

// used to configure the user's environment.
type me struct {
	global *global
}

func (t *me) configure(parent *kingpin.CmdClause) {
	parent.Command("show", "show current credentials").Action(t.show)
	parent.Command("init", "initialize the users credentials").Action(t.exec)
}

func (t *me) show(ctx *kingpin.ParseContext) (err error) {
	var (
		print string
		pub   []byte
	)
	if print, pub, err = notary.AutoSignerInfo(); err != nil {
		return err
	}

	log.Printf("fingerprint: %s\npublic key:\n%s\n", print, string(pub))

	return nil
}

func (t *me) exec(ctx *kingpin.ParseContext) (err error) {
	if _, err = notary.NewAutoSigner(bw.DisplayName()); err != nil {
		return err
	}

	return nil
}
