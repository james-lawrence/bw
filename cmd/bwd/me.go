package main

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/alecthomas/kingpin"
	"github.com/james-lawrence/bw"
	"github.com/james-lawrence/bw/agent"
	"github.com/james-lawrence/bw/cmd/commandutils"
	"github.com/james-lawrence/bw/internal/x/debugx"
	"github.com/james-lawrence/bw/notary"
	"github.com/pkg/errors"
)

// used to configure the user's environment.
type me struct {
	global      *global
	environment string
}

func (t *me) configure(parent *kingpin.CmdClause) {
	parent.Command("show", "show current credentials").Default().Action(t.show)
	parent.Command("pub", "print public key to stdout").Action(t.pubkey)
	cmd := parent.Command("init", "initialize the users credentials for a workspace").Action(t.exec)
	cmd.Arg("environment", "environment to insert the authorized key for deployment").Default(bw.DefaultEnvironmentName).StringVar(&t.environment)
	parent.Command("clear", "remove the current credentials from disk").Action(t.reset)
}

func (t *me) show(ctx *kingpin.ParseContext) (err error) {
	var (
		print string
		pub   []byte
	)
	if print, pub, err = notary.AutoSignerInfo(); err != nil {
		return err
	}

	fmt.Println("location:", notary.PublicKeyPath())
	fmt.Println("fingerprint:", print)
	fmt.Println("public key:", string(pub))

	return nil
}

func (t *me) pubkey(ctx *kingpin.ParseContext) (err error) {
	var (
		pub []byte
	)

	if _, pub, err = notary.AutoSignerInfo(); err != nil {
		return err
	}

	fmt.Println(string(pub))

	return nil
}

func (t *me) exec(ctx *kingpin.ParseContext) (err error) {
	var (
		config      agent.ConfigClient
		fingerprint string
		encoded     []byte
	)

	if _, err = notary.NewAutoSigner(bw.DisplayName()); err != nil {
		return err
	}

	if config, err = commandutils.ReadConfiguration(t.environment); err != nil {
		if os.IsNotExist(errors.Cause(err)) {
			debugx.Println("init failed to locate a valid environment", err)
			return nil
		}
		return err
	}

	if fingerprint, encoded, err = notary.AutoSignerInfo(); err != nil {
		return err
	}

	return notary.ReplaceAuthorizedKey(
		filepath.Join(config.Dir(), bw.AuthKeysFile),
		fingerprint,
		encoded,
	)
}

func (t *me) reset(ctx *kingpin.ParseContext) (err error) {
	return notary.ClearAutoSignerKey()
}
