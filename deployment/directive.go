package deployment

import (
	"os"
	"path/filepath"
	"strings"

	"github.com/james-lawrence/bw"
	"github.com/james-lawrence/bw/directives"
	"github.com/james-lawrence/bw/directives/shell"
	"github.com/james-lawrence/bw/internal/x/errorsx"
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
func (t Directive) Deploy(dctx *DeployContext) {
	for _, opt := range t.options {
		opt(&t)
	}

	go t.deploy(dctx)
}

func (t Directive) deploy(dctx *DeployContext) {
	var (
		err      error
		dinterp  directives.InterpLoader
		dfs      directives.ArchiveLoader
		dshell   directives.ShellLoader
		loaded   []directives.Loaded
		environ  []string
		tmpdir   string
		cachedir string
	)

	if environ, err = shell.EnvironFromFile(filepath.Join(dctx.ArchiveRoot, bw.EnvFile)); err != nil {
		dctx.Done(err)
		return
	}

	if tmpdir, err = mkdirTemp(dctx.TempRoot, ".bw-tmp-*"); err != nil {
		dctx.Done(err)
		return
	}
	done := func(cause error) {
		dctx.Done(errorsx.Compact(err, os.RemoveAll(tmpdir)))
	}

	cachedir = filepath.Join(dctx.CacheRoot, bw.DirCache)
	if err = os.MkdirAll(cachedir, 0750); err != nil {
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
			shell.OptionDeployID(dctx.ID.String()),
			shell.OptionLogger(dctx.Log),
			shell.OptionEnviron(append(t.sctx.Environ, environ...)),
			shell.OptionDir(dctx.ArchiveRoot),
			shell.OptionTempDir(tmpdir),
			shell.OptionCacheDir(cachedir),
		),
	}

	dfs = directives.ArchiveLoader{
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
		dfs,
		directives.NewAWSELBAttach(),
		directives.NewAWSELBDetach(),
		directives.NewAWSELB2Attach(),
		directives.NewAWSELB2Detach(),
	}

	dctx.Log.Println("---------------------- DURATION", dctx.timeout(), "----------------------")
	root := filepath.Join(dctx.ArchiveRoot, t.directory)
	if loaded, err = directives.Load(dctx.Log, root, loaders...); err != nil {
		dctx.Dispatch()
		done(errors.Wrapf(err, "failed to load directives"))
		return
	}

	dctx.Log.Println("loaded", len(loaded), "directive(s) from", root)
	for _, l := range loaded {
		name := strings.TrimPrefix(l.Path, root+"/")
		dctx.Log.Println("initiated directive:", name)
		if err = l.Run(dctx.deadline); err != nil {
			dctx.Log.Println("failed directive:", name, err)
			done(err)
			return
		}
		dctx.Log.Println("completed directive:", name)
	}
	done(err)
}
