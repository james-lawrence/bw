package main

import (
	"io/ioutil"
	"log"
	"os"
	"path/filepath"

	"github.com/alecthomas/kingpin"
	"github.com/james-lawrence/bw/agent"
	"github.com/pkg/errors"
	yaml "gopkg.in/yaml.v2"

	"github.com/james-lawrence/bw"
)

type environmentCmd struct {
	global *global
}

func (t *environmentCmd) configure(parent *kingpin.CmdClause) {
	(&environmentCreate{global: t.global}).configure(parent.Command("create", "initialize an environment"))
}

type environmentCreate struct {
	global      *global
	name        string
	path        string
	dialAddress string
}

func (t *environmentCreate) configure(parent *kingpin.CmdClause) {
	parent.Flag("directory", "path of the environment directory to create").Default(bw.DefaultDeployspaceConfigDir).StringVar(&t.path)
	parent.Arg("name", "name of the environment being created").Required().StringVar(&t.name)
	parent.Arg("address", "address to dial when connecting to this environment").Required().StringVar(&t.dialAddress)
	parent.Action(t.generate)
}

func (t *environmentCreate) generate(ctx *kingpin.ParseContext) (err error) {
	type context struct {
		Address string
	}

	var (
		encoded []byte
		cc      agent.ConfigClient
	)

	if err = errors.WithStack(os.MkdirAll(filepath.Join(t.path, t.name), 0755)); err != nil {
		return err
	}

	cc = agent.ExampleConfigClient(
		agent.CCOptionAddress(t.dialAddress),
		agent.CCOptionConcurrency(1),
		agent.CCOptionTLSConfig(t.name),
		agent.CCOptionEnvironment("FOO=BAR\n"),
	)

	if encoded, err = yaml.Marshal(cc); err != nil {
		return errors.Wrap(err, "failed to encode configuration, try updating to a newer bearded wookie version")
	}

	if err = ioutil.WriteFile(filepath.Join(t.path, t.name, bw.DefaultClientConfig), encoded, 0600); err != nil {
		return errors.Wrap(err, "failed to write configuration")
	}

	if err = ioutil.WriteFile(filepath.Join(t.path, t.name, bw.AuthKeysFile), []byte(nil), 0600); err != nil {
		return errors.Wrap(err, "failed to write configuration")
	}

	environmentCredentialsDir := bw.DefaultLocation(t.name, "")
	if os.Geteuid() > 0 {
		environmentCredentialsDir = bw.DefaultUserDirLocation(t.name)
	}

	log.Println("creating configuration directory:", environmentCredentialsDir)
	if err = os.MkdirAll(environmentCredentialsDir, 0700); err != nil {
		return errors.WithStack(err)
	}

	return nil
}
