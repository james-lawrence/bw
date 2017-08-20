package agent

import (
	"encoding/hex"
	"log"
	"path/filepath"

	"github.com/pkg/errors"

	"bitbucket.org/jatone/bearded-wookie/agentutil"
	"bitbucket.org/jatone/bearded-wookie/archive"
	"bitbucket.org/jatone/bearded-wookie/deployment/agent"
	"bitbucket.org/jatone/bearded-wookie/directives"
	"bitbucket.org/jatone/bearded-wookie/directives/shell"
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

// NewDirective builds a coordinator
func NewDirective(options ...DirectiveOption) Directive {
	d := Directive{
		keepN:   3,
		options: options,
	}

	return d
}

// Directive ...
type Directive struct {
	keepN   int
	root    string
	archive agent.Archive
	sctx    shell.Context
	options []DirectiveOption
}

// Deploy ...
func (t Directive) Deploy(archive *agent.Archive, completed chan error) error {
	log.Printf("deploy recieved: deployID(%s) leader(%s) location(%s)\n", hex.EncodeToString(archive.DeploymentID), archive.Leader, archive.Location)
	defer log.Printf("deploy complete: deployID(%s) leader(%s) location(%s)\n", hex.EncodeToString(archive.DeploymentID), archive.Leader, archive.Location)

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
		_directives []directives.Directive
	)
	log.Println("deploying")
	defer log.Println("deploy complete")

	dshell := directives.ShellLoader{
		Context: t.sctx,
	}

	dpkg := directives.PackageLoader{}
	dst := filepath.Join(t.root, hex.EncodeToString(t.archive.DeploymentID))

	if err = archive.Unpack(dst, NewDownloader(t.archive.Location).Download()); err != nil {
		err = errors.Wrapf(err, "retrieve archive")
		goto done
	}

	log.Println("workspace downloaded", dst)

	if _directives, err = directives.Load(filepath.Join(dst, ".directives"), dshell, dpkg); err != nil {
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
	// cleanup workspace directory.
	if soft := agentutil.MaybeClean(agentutil.KeepNewestN(t.keepN))(agentutil.Dirs(t.root)); soft != nil {
		log.Println("failed to clean workspace directory", soft)
	}
	completed <- err
}
