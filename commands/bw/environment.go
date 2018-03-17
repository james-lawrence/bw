package main

import (
	"html/template"
	"log"
	"os"
	"path/filepath"

	"github.com/alecthomas/kingpin"
	"github.com/pkg/errors"

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
	const (
		skeletonEnvironment = `address: "{{.Address}}:2000"
servername: "{{.Address}}"
deployTimeout: "3m"
concurrency: 0.2
environment: |
	BEARDED_WOOKIE_EXAMPLE=ENVIRONMENT_VARIABLE
`
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

	environmentCredentialsDir := bw.DefaultLocation(t.name, "")
	if os.Geteuid() > 0 {
		environmentCredentialsDir = bw.DefaultUserDirLocation(t.name, "")
	}

	log.Println("creating configuration directory:", environmentCredentialsDir)
	if err = os.MkdirAll(environmentCredentialsDir, 0700); err != nil {
		return errors.WithStack(err)
	}

	return nil
}
