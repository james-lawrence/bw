package deployment

import (
	"path/filepath"

	"github.com/james-lawrence/bw"
	"github.com/james-lawrence/bw/directives"
	"github.com/james-lawrence/bw/directives/shell"
	"github.com/pkg/errors"
)

// Well known directory names.
const (
	LocalDirName  = ".local"
	RemoteDirName = ".remote"
)

// DirectiveOption ...
type DirectiveOption func(*Directive)

// DirectiveOptionShellContext ...
func DirectiveOptionShellContext(ctx shell.Context) DirectiveOption {
	return func(d *Directive) {
		d.sctx = ctx
	}
}

// DirectiveOptionDir specify what subdirect directory to load directives from within
// the archive.
func DirectiveOptionDir(dir string) DirectiveOption {
	return func(d *Directive) {
		d.directory = dir
	}
}

// NewDirective builds a coordinator
func NewDirective(options ...DirectiveOption) Directive {
	d := Directive{
		directory: RemoteDirName,
		options:   options,
	}

	return d
}

// Directive ...
type Directive struct {
	sctx      shell.Context
	directory string
	options   []DirectiveOption
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
		dinterp directives.InterpLoader
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

	dinterp = directives.InterpLoader{
		Context:      dc,
		Environ:      dshell.Context.Environ,
		ShellContext: dshell.Context,
	}

	loaders := []directives.Loader{
		dshell,
		dinterp,
		dpkg,
		dfs,
		directives.NewAWSELBAttach(),
		directives.NewAWSELBDetach(),
		directives.NewAWSELB2Attach(),
		directives.NewAWSELB2Detach(),
	}

	dctx.Log.Println("---------------------- DURATION", dctx.timeout(), "----------------------")
	if d, err = directives.Load(dctx.Log, filepath.Join(dctx.ArchiveRoot, t.directory), loaders...); err != nil {
		dctx.Dispatch()
		dctx.Done(errors.Wrapf(err, "failed to load directives"))
		return
	}

	dctx.Log.Println("loaded", len(d), "directive(s)")
	for _, l := range d {
		dctx.Log.Println("running directive")
		if err = l.Run(dctx.deadline); err != nil {
			dctx.Log.Println("directive failed", err)
			dctx.Done(err)
			return
		}
	}

	dctx.Done(err)
}
