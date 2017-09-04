package agent

import (
	"log"
	"path/filepath"

	"bitbucket.org/jatone/bearded-wookie"
	"bitbucket.org/jatone/bearded-wookie/agentutil"
	"bitbucket.org/jatone/bearded-wookie/archive"
	"bitbucket.org/jatone/bearded-wookie/deployment"
	"bitbucket.org/jatone/bearded-wookie/deployment/agent"
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

// DirectiveOptionArchive ...
func DirectiveOptionArchive(archive agent.Archive) DirectiveOption {
	return func(d *Directive) {
		d.archive = archive
	}
}

// DirectiveOptionRoot ...
func DirectiveOptionRoot(root string) DirectiveOption {
	return func(d *Directive) {
		d.root = root
	}
}

// DirectiveOptionKeepN the number of previous deploys to keep.
func DirectiveOptionKeepN(n int) DirectiveOption {
	return func(d *Directive) {
		d.keepN = n
	}
}

// NewDirective builds a coordinator
func NewDirective(options ...DirectiveOption) Directive {
	d := Directive{
		keepN:   3,
		options: options,
		dlreg:   downloads.New(),
	}

	return d
}

// Directive ...
type Directive struct {
	keepN   int
	root    string
	archive agent.Archive
	sctx    shell.Context
	dlreg   downloads.Registry
	options []DirectiveOption
}

// Deploy ...
func (t Directive) Deploy(archive *agent.Archive, completed chan error) error {
	log.Printf("deploy recieved: deployID(%s) leader(%s) location(%s)\n", bw.RandomID(archive.DeploymentID), archive.Leader, archive.Location)
	defer log.Printf("deploy complete: deployID(%s) leader(%s) location(%s)\n", bw.RandomID(archive.DeploymentID), archive.Leader, archive.Location)

	options := append(
		t.options,
		DirectiveOptionArchive(*archive),
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
		dctx        deployment.DeployContext
		dshell      directives.ShellLoader
		dpkg        directives.PackageLoader
		dst         string
		_directives []directives.Directive
	)

	log.Println("deploying")
	defer log.Println("deploy complete")

	if dctx, err = deployment.NewDeployContext(t.root, t.archive); err != nil {
		goto done
	}

	dshell = directives.ShellLoader{
		Context: shell.NewContext(t.sctx, shell.OptionLogger(dctx.Log)),
	}

	dpkg = directives.PackageLoader{}
	dst = filepath.Join(t.root, bw.RandomID(t.archive.DeploymentID).String(), "archive")

	dctx.Log.Println("attempting to download", t.archive.Location)

	if err = archive.Unpack(dst, t.dlreg.New(t.archive.Location).Download()); err != nil {
		err = errors.Wrapf(err, "retrieve archive")
		goto done
	}

	dctx.Log.Println("completed download", dst)

	if _directives, err = directives.Load(filepath.Join(dst, ".remote"), dshell, dpkg); err != nil {
		err = errors.Wrapf(err, "failed to load directives")
		goto done
	}

	dctx.Log.Println("loaded", len(_directives), "directive(s)")
	for _, l := range _directives {
		if err = l.Run(); err != nil {
			goto done
		}
	}

done:
	// cleanup workspace directory.
	if soft := agentutil.MaybeClean(agentutil.KeepNewestN(t.keepN))(agentutil.Dirs(t.root)); soft != nil {
		dctx.Log.Println("failed to clean workspace directory", soft)
	}
	completed <- dctx.Done(err)
}
