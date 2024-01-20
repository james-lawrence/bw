package main

import (
	"fmt"
	"path/filepath"

	"github.com/james-lawrence/bw"
	"github.com/james-lawrence/bw/agent"
	"github.com/james-lawrence/bw/cmd/bw/cmdopts"
	"github.com/james-lawrence/bw/cmd/commandutils"
	"github.com/james-lawrence/bw/internal/stringsx"
	"github.com/james-lawrence/bw/notary"
	"github.com/james-lawrence/bw/vcsinfo"
)

// used to configure the user's environment.
type cmdMe struct {
	Show  cmdMeShow  `cmd:"" help:"show profile credentials"`
	Pub   cmdMePub   `cmd:"" help:"print public key to stdout"`
	Init  cmdMeInit  `cmd:"" help:"initialize the user's credentials for a workspace"`
	Clear cmdMeClear `cmd:"" help:"remove the current credentials from disk"`
}

type cmdMeShow struct{}

func (t cmdMeShow) Run(ctx *cmdopts.Global) (err error) {
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

type cmdMePub struct {
}

func (t cmdMePub) Run(ctx *cmdopts.Global) (err error) {
	var (
		pub []byte
	)

	if _, pub, err = notary.AutoSignerInfo(); err != nil {
		return err
	}

	fmt.Println(string(pub))

	return nil
}

type cmdMeInit struct {
	cmdopts.BeardedWookieEnv
	Name string `help:"name to assign to the comment"`
}

func (t cmdMeInit) Run(ctx *cmdopts.Global) (err error) {
	var (
		config      agent.ConfigClient
		fingerprint string
		encoded     []byte
	)

	if config, err = commandutils.ReadConfiguration(t.Environment); err != nil {
		return err
	}

	displayname := vcsinfo.CurrentUserDisplay(config.WorkDir())

	if _, err = notary.NewAutoSigner(stringsx.DefaultIfBlank(t.Name, displayname)); err != nil {
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

type cmdMeClear struct{}

func (t cmdMeClear) Run(ctx *cmdopts.Global) error {
	return notary.ClearAutoSignerKey()
}
