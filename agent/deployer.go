package agent

import (
	"log"
	"path/filepath"

	"bitbucket.org/jatone/bearded-wookie/archive"
	"bitbucket.org/jatone/bearded-wookie/deployment"
	"bitbucket.org/jatone/bearded-wookie/directives"
	"bitbucket.org/jatone/bearded-wookie/directives/shell"
	"bitbucket.org/jatone/bearded-wookie/downloads"
	"github.com/pkg/errors"
)

// DirectiveOption ...
type DirectiveOption func(*Directive)

// DirectiveOptionShellContext ...
func DirectiveOptionShellContext(ctx shell.Context) DirectiveOption {
	return func(d *Directive) {
		d.sctx = ctx
	}
}

// DirectiveOptionDeployContext ...
func DirectiveOptionDeployContext(dctx deployment.DeployContext) DirectiveOption {
	return func(d *Directive) {
		d.dctx = dctx
	}
}

// NewDirective builds a coordinator
func NewDirective(options ...DirectiveOption) Directive {
	d := Directive{
		options: options,
		dlreg:   downloads.New(),
	}

	return d
}

// Directive ...
type Directive struct {
	dctx    deployment.DeployContext
	sctx    shell.Context
	dlreg   downloads.Registry
	options []DirectiveOption
}

// Deploy ...
func (t Directive) Deploy(dctx deployment.DeployContext, completed chan error) error {
	log.Printf("deploy recieved: deployID(%s) leader(%s) location(%s)\n", dctx.ID, dctx.Archive.Leader, dctx.Archive.Location)
	defer log.Printf("deploy complete: deployID(%s) leader(%s) location(%s)\n", dctx.ID, dctx.Archive.Leader, dctx.Archive.Location)

	options := append(
		t.options,
		DirectiveOptionDeployContext(dctx),
	)

	for _, opt := range options {
		opt(&t)
	}

	go t.deploy(completed)

	return nil
}

func (t Directive) deploy(completed chan error) {
	var (
		err         error
		dshell      directives.ShellLoader
		dpkg        directives.PackageLoader
		dst         string
		_directives []directives.Directive
	)

	log.Println("deploying")
	defer log.Println("deploy complete")

	dshell = directives.ShellLoader{
		Context: shell.NewContext(t.sctx, shell.OptionLogger(t.dctx.Log)),
	}

	dpkg = directives.PackageLoader{}
	dst = filepath.Join(t.dctx.Root, "archive")

	t.dctx.Log.Println("attempting to download", t.dctx.Archive.Location)

	if err = archive.Unpack(dst, t.dlreg.New(t.dctx.Archive.Location).Download()); err != nil {
		err = errors.Wrapf(err, "retrieve archive")
		goto done
	}

	t.dctx.Log.Println("completed download", dst)

	if _directives, err = directives.Load(filepath.Join(dst, ".remote"), dshell, dpkg); err != nil {
		err = errors.Wrapf(err, "failed to load directives")
		goto done
	}

	t.dctx.Log.Println("loaded", len(_directives), "directive(s)")
	for _, l := range _directives {
		if err = l.Run(); err != nil {
			goto done
		}
	}

done:
	completed <- t.dctx.Done(err)
}