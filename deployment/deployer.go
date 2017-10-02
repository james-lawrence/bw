package deployment

import (
	"path/filepath"

	"bitbucket.org/jatone/bearded-wookie/archive"
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
func DirectiveOptionDeployContext(dctx DeployContext) DirectiveOption {
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
	dctx    DeployContext
	sctx    shell.Context
	dlreg   downloads.Registry
	options []DirectiveOption
}

// Deploy ...
func (t Directive) Deploy(dctx DeployContext) error {
	options := append(
		t.options,
		DirectiveOptionDeployContext(dctx),
	)

	for _, opt := range options {
		opt(&t)
	}

	go t.deploy()

	return nil
}

func (t Directive) deploy() {
	var (
		err    error
		dshell directives.ShellLoader
		dpkg   directives.PackageLoader
		dst    string
		d      []directives.Directive
	)

	t.dctx.Log.Printf("deploy recieved: deployID(%s) leader(%s) location(%s)\n", t.dctx.ID, t.dctx.Archive.Peer.Name, t.dctx.Archive.Location)
	defer t.dctx.Log.Printf("deploy complete: deployID(%s) leader(%s) location(%s)\n", t.dctx.ID, t.dctx.Archive.Peer.Name, t.dctx.Archive.Location)

	dshell = directives.ShellLoader{
		Context: shell.NewContext(t.sctx, shell.OptionLogger(t.dctx.Log)),
	}

	dpkg = directives.PackageLoader{}
	dst = filepath.Join(t.dctx.Root, "archive")

	t.dctx.Log.Println("attempting to download", t.dctx.Archive.Location)

	if err = errors.Wrapf(archive.Unpack(dst, t.dlreg.New(t.dctx.Archive.Location).Download()), "retrieve archive"); err != nil {
		t.dctx.Done(err)
		return
	}

	t.dctx.Log.Println("completed download", dst)

	if d, err = directives.Load(t.dctx.Log, filepath.Join(dst, ".remote"), dshell, dpkg); err != nil {
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
