package main

import (
	"log"
	"os"
	"path/filepath"

	"github.com/bits-and-blooms/bloom/v3"
	"github.com/davecgh/go-spew/spew"
	"github.com/james-lawrence/bw/agent"
	"github.com/james-lawrence/bw/cmd/bw/cmdopts"
	"github.com/james-lawrence/bw/cmd/commandutils"
	"github.com/james-lawrence/bw/notary"
	"github.com/pkg/errors"
	yaml "gopkg.in/yaml.v2"

	"github.com/james-lawrence/bw"
)

type cmdEnv struct {
	Create cmdEnvCreate `cmd:"" help:"initialize an environment"`
	Users  cmdEnvUsers  `cmd:"" help:"list the users and their permissions in the environments bw.auth.keys file"`
}

type cmdEnvCreate struct {
	Directory string `help:"path of the environment directory to create" default:"${vars_bw_default_deployspace_config_directory}"`
	Name      string `help:"name of the environment being created"`
	Address   string `help:"address to dial when connecting to this environment"`
}

func (t *cmdEnvCreate) Run(ctx *cmdopts.Global) (err error) {
	var (
		encoded []byte
		cc      agent.ConfigClient
	)

	if err = errors.WithStack(os.MkdirAll(filepath.Join(t.Directory, t.Name), 0755)); err != nil {
		return err
	}

	cc = agent.ExampleConfigClient(
		agent.CCOptionAddress(t.Address),
		agent.CCOptionConcurrency(1),
		agent.CCOptionTLSConfig(t.Name),
		agent.CCOptionEnvironment("FOO=BAR\n"),
	)

	if encoded, err = yaml.Marshal(cc); err != nil {
		return errors.Wrap(err, "failed to encode configuration, try updating to a newer bearded wookie version")
	}

	if err = os.WriteFile(filepath.Join(t.Directory, t.Name, bw.DefaultClientConfig), encoded, 0600); err != nil {
		return errors.Wrap(err, "failed to write configuration")
	}

	if err = os.WriteFile(filepath.Join(t.Directory, t.Name, bw.AuthKeysFile), []byte(nil), 0600); err != nil {
		return errors.Wrap(err, "failed to write configuration")
	}

	environmentCredentialsDir := bw.DefaultLocation(t.Name, "")
	if os.Geteuid() > 0 {
		environmentCredentialsDir = bw.DefaultUserDirLocation(t.Name)
	}

	log.Println("creating configuration directory:", environmentCredentialsDir)
	if err = os.MkdirAll(environmentCredentialsDir, 0700); err != nil {
		return errors.WithStack(err)
	}

	return nil
}

type cmdEnvUsers struct {
	cmdopts.BeardedWookieEnv
}

func (t *cmdEnvUsers) Run(ctx *cmdopts.Global) (err error) {
	var (
		n      notary.Composite
		config agent.ConfigClient
	)

	if config, err = commandutils.ReadConfiguration(t.Environment); err != nil {
		return err
	}

	if n, err = notary.NewFromFile(config.Dir(), bw.AuthKeysFile); err != nil {
		return err
	}
	b := bloom.NewWithEstimates(1000, 0.0001)

	out := make(chan *notary.Grant, 200)
	errc := make(chan error)
	go func() {
		select {
		case errc <- n.Sync(ctx.Context, b, out):
		case <-ctx.Context.Done():
			errc <- ctx.Context.Err()
		}
	}()

	for {
		select {
		case g := <-out:
			log.Println(g.Fingerprint, spew.Sdump(g.Permission))
		case err := <-errc:
			return err
		}
	}
}
