package deployment

import (
	"path/filepath"

	"github.com/james-lawrence/bw"
	"github.com/james-lawrence/bw/directives"
	"github.com/james-lawrence/bw/directives/shell"
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

// NewDirective builds a coordinator
func NewDirective(options ...DirectiveOption) Directive {
	d := Directive{
		options: options,
	}

	return d
}

// Directive ...
type Directive struct {
	sctx    shell.Context
	options []DirectiveOption
}

// Deploy ...
func (t Directive) Deploy(dctx DeployContext) {
	for _, opt := range t.options {
		opt(&t)
	}

	go t.deploy(dctx)
}

func (t Directive) deploy(dctx DeployContext) {
	var (
		err     error
		dfs     directives.ArchiveLoader
		dshell  directives.ShellLoader
		dpkg    directives.PackageLoader
		d       []directives.Directive
		environ []string
	)

	if environ, err = shell.EnvironFromFile(filepath.Join(dctx.ArchiveRoot, bw.EnvFile)); err != nil {
		dctx.Done(err)
		return
	}

	dc := directives.Context{
		RootDirectory: dctx.Root,
		Log:           dctx.Log,
	}

	dshell = directives.ShellLoader{
		Context: shell.NewContext(
			t.sctx,
			shell.OptionLogger(dctx.Log),
			shell.OptionEnviron(append(t.sctx.Environ, environ...)),
			shell.OptionDir(dctx.ArchiveRoot),
		),
	}

	dfs = directives.ArchiveLoader{
		Context: dc,
	}

	dpkg = directives.PackageLoader{
		Context: dc,
	}

	loaders := []directives.Loader{
		dshell,
		dpkg,
		dfs,
		directives.NewAWSELBAttach(),
		directives.NewAWSELBDetach(),
		directives.NewAWSELB2Attach(),
		directives.NewAWSELB2Detach(),
	}

	if d, err = directives.Load(dctx.Log, filepath.Join(dctx.ArchiveRoot, ".remote"), loaders...); err != nil {
		dctx.Done(errors.Wrapf(err, "failed to load directives"))
		return
	}

	dctx.Log.Println("loaded", len(d), "directive(s)")
	for _, l := range d {
		if err = l.Run(); err != nil {
			dctx.Done(err)
			return
		}
	}

	dctx.Done(err)
}
