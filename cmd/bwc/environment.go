package main

import (
	"io/ioutil"
	"log"
	"os"
	"path/filepath"

	"github.com/james-lawrence/bw/agent"
	"github.com/james-lawrence/bw/cmd/bwc/cmdopts"
	"github.com/pkg/errors"
	yaml "gopkg.in/yaml.v2"

	"github.com/james-lawrence/bw"
)

type cmdEnv struct {
	Create cmdEnvCreate `cmd:"" help:"initialize an environment"`
}

type cmdEnvCreate struct {
	Directory string `help:"path of the environment directory to create" default:"${vars.bw.default.deployspace.config.directory}"`
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

	if err = ioutil.WriteFile(filepath.Join(t.Directory, t.Name, bw.DefaultClientConfig), encoded, 0600); err != nil {
		return errors.Wrap(err, "failed to write configuration")
	}

	if err = ioutil.WriteFile(filepath.Join(t.Directory, t.Name, bw.AuthKeysFile), []byte(nil), 0600); err != nil {
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
