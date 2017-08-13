package deployment

import (
	"log"
	"path/filepath"

	"github.com/pkg/errors"

	"bitbucket.org/jatone/bearded-wookie/directives"
	"bitbucket.org/jatone/bearded-wookie/directives/shell"
)

// DirectiveOption ...
type DirectiveOption func(*Directive)

// DirectiveOptionBaseDirectory ...
func DirectiveOptionBaseDirectory(dir string) DirectiveOption {
	return func(d *Directive) {
		d.baseDir = dir
	}
}

// DirectiveOptionShellContext ...
func DirectiveOptionShellContext(ctx shell.Context) DirectiveOption {
	return func(d *Directive) {
		d.sctx = ctx
	}
}

// NewPackagekit builds a coordinator that uses packagekit to install packages.
func NewDirective(options ...DirectiveOption) Directive {
	d := Directive{
		options: options,
	}

	return d
}

type Directive struct {
	baseDir string
	sctx    shell.Context
	options []DirectiveOption
}

func (t Directive) Deploy(completed chan error) error {
	for _, opt := range t.options {
		opt(&t)
	}

	go t.deploy(completed)

	return nil
}

func (t Directive) deploy(completed chan error) {
	var (
		err         error
		_directives []directives.Directive
	)

	log.Println("deploying")
	defer log.Println("deploy complete")

	dshell := directives.ShellLoader{
		Context: t.sctx,
	}
	dpkg := directives.PackageLoader{}

	if _directives, err = directives.Load(filepath.Join(t.baseDir, ".directives"), dshell, dpkg); err != nil {
		err = errors.Wrapf(err, "failed to load directives")
		goto done
	}

	log.Println("loaded", len(_directives), "directive(s)")
	for _, l := range _directives {
		if err = l.Run(); err != nil {
			goto done
		}
	}
done:
	completed <- err
}
