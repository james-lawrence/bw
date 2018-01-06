package deployment

import (
	"path/filepath"

	"github.com/james-lawrence/bw"
	"github.com/james-lawrence/bw/archive"
	"github.com/james-lawrence/bw/directives"
	"github.com/james-lawrence/bw/directives/dynplugin"
	"github.com/james-lawrence/bw/directives/shell"
	"github.com/james-lawrence/bw/storage"
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
func DirectiveOptionDeployContext(dctx DeployContext) DirectiveOption {
	return func(d *Directive) {
		d.dctx = dctx
	}
}

// DirectiveOptionPlugins ...
func DirectiveOptionPlugins(p ...dynplugin.Directive) DirectiveOption {
	return func(d *Directive) {
		d.plugins = p
	}
}

// DirectiveOptionDownloadRegistry ...
func DirectiveOptionDownloadRegistry(reg storage.Registry) DirectiveOption {
	return func(d *Directive) {
		d.dlreg = reg
	}
}

// NewDirective builds a coordinator
func NewDirective(options ...DirectiveOption) Directive {
	d := Directive{
		options: options,
		dlreg:   storage.New(),
	}

	return d
}

// Directive ...
type Directive struct {
	dctx    DeployContext
	sctx    shell.Context
	plugins []dynplugin.Directive
	dlreg   storage.Registry
	options []DirectiveOption
}

// Deploy ...
func (t Directive) Deploy(dctx DeployContext) {
	options := append(
		t.options,
		DirectiveOptionDeployContext(dctx),
	)

	for _, opt := range options {
		opt(&t)
	}

	go t.deploy()
}

func (t Directive) deploy() {
	var (
		err     error
		dfs     directives.ArchiveLoader
		dshell  directives.ShellLoader
		dpkg    directives.PackageLoader
		dst     string
		d       []directives.Directive
		environ []string
	)

	t.dctx.Log.Printf("deploy recieved: deployID(%s) leader(%s) location(%s)\n", t.dctx.ID, t.dctx.Archive.Peer.Name, t.dctx.Archive.Location)
	defer t.dctx.Log.Printf("deploy complete: deployID(%s) leader(%s) location(%s)\n", t.dctx.ID, t.dctx.Archive.Peer.Name, t.dctx.Archive.Location)

	dst = filepath.Join(t.dctx.Root, "archive")
	t.dctx.Log.Println("attempting to download", t.dctx.Archive.Location)

	if err = errors.Wrapf(archive.Unpack(dst, t.dlreg.New(t.dctx.Archive.Location).Download()), "retrieve archive"); err != nil {
		t.dctx.Done(err)
		return
	}

	t.dctx.Log.Println("completed download", dst)

	if environ, err = shell.EnvironFromFile(filepath.Join(dst, bw.EnvFile)); err != nil {
		t.dctx.Done(err)
		return
	}

	dc := directives.Context{
		RootDirectory: t.dctx.Root,
		Log:           t.dctx.Log,
	}

	dshell = directives.ShellLoader{
		Context: shell.NewContext(
			t.sctx,
			shell.OptionLogger(t.dctx.Log),
			shell.OptionEnviron(append(t.sctx.Environ, environ...)),
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

	for _, p := range t.plugins {
		loaders = append(loaders, p)
	}

	if d, err = directives.Load(t.dctx.Log, filepath.Join(dst, ".remote"), loaders...); err != nil {
		t.dctx.Done(errors.Wrapf(err, "failed to load directives"))
		return
	}

	t.dctx.Log.Println("loaded", len(d), "directive(s)")
	for _, l := range d {
		if err = l.Run(); err != nil {
			t.dctx.Done(err)
			return
		}
	}

	t.dctx.Done(err)
}
