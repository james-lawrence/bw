package main

import (
	"html/template"
	"os"
	"path/filepath"

	"github.com/alecthomas/kingpin"
	"github.com/pkg/errors"

	"bitbucket.org/jatone/bearded-wookie"
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
	parent.Flag("name", "name of the environment to create").Default(bw.DefaultEnvironmentName).StringVar(&t.name)
	parent.Arg("address", "address to dial when connecting to this environment").Required().StringVar(&t.dialAddress)
	parent.Action(t.generate)
}

func (t *environmentCreate) generate(ctx *kingpin.ParseContext) (err error) {
	type context struct {
		Address string
	}
	const (
		skeletonEnvironment = `address: "{{.Address}}:2000"
tlsconfig:
    servername: "{{.Address}}"
deploymentconfig:
    strategy: percent
    options:
    percentage: 0.5`
	)
	var (
		dst *os.File
	)

	if err = errors.WithStack(os.MkdirAll(t.path, 0755)); err != nil {
		return err
	}

	if dst, err = os.Create(filepath.Join(t.path, t.name)); err != nil {
		return errors.WithStack(err)
	}
	defer dst.Close()

	if err = template.Must(template.New("").Parse(skeletonEnvironment)).Execute(dst, context{Address: t.dialAddress}); err != nil {
		return errors.WithStack(err)
	}

	return nil
}
