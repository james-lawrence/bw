package main

import (
	"io/ioutil"
	"os"
	"path/filepath"

	"github.com/alecthomas/kingpin"
	"github.com/pkg/errors"

	"github.com/james-lawrence/bw"
)

type workspaceCmd struct {
	global *global
}

func (t *workspaceCmd) configure(parent *kingpin.CmdClause) {
	(&workspaceCreate{global: t.global}).configure(parent.Command("create", "initialize a workspace"))
}

type workspaceCreate struct {
	global          *global
	path            string
	includeExamples bool
}

func (t *workspaceCreate) configure(parent *kingpin.CmdClause) {
	parent.Arg("directory", "path of the workspace directory to create").Default(bw.DefaultDeployspaceDir).StringVar(&t.path)
	parent.Flag("examples", "include examples").Default("true").BoolVar(&t.includeExamples)
	parent.Action(t.generate)
}

func (t *workspaceCreate) generate(ctx *kingpin.ParseContext) (err error) {
	const (
		skeletonShellDirective = `- command: "echo hello world"
- command: "echo hello ${USER}"
- command: "echo 'Hostname(%H) | Machine ID(%m) | DN(%d) | FQDN(%f) | Username(%u) | User ID(%U) | Homedir(%h) | %%'" # substitution examples.
- command: "/usr/bin/false"
  lenient: true # allows the command to fail.
- command: "/usr/bin/sleep 15"
  timeout: 10s`
		skeletonRestartDetach = `// will detach the instance from any elb loadbalancers as determined by the autoscaling group of the instance`
		skeletonRestartAttach = `// will attach the instance to any elb loadbalancers as determined by the autoscaling group of the instance`
		skeletonRestart       = `- command: "echo restart application"`
		skeletonFinal         = `- command: "echo deploy complete"`
	)

	if err = errors.WithStack(os.MkdirAll(t.path, 0755)); err != nil {
		return err
	}

	if err = errors.WithStack(os.MkdirAll(filepath.Join(t.path, ".remote"), 0755)); err != nil {
		return err
	}

	if t.includeExamples {
		if err = ioutil.WriteFile(filepath.Join(t.path, ".remote", "01_shell.bwcmd"), []byte(skeletonShellDirective), 0600); err != nil {
			return errors.WithStack(err)
		}

		if err = errors.WithStack(os.MkdirAll(filepath.Join(t.path, ".remote", "02_restart_module"), 0755)); err != nil {
			return err
		}

		if err = ioutil.WriteFile(filepath.Join(t.path, ".remote", "02_restart_module", "00_pre_restart.detach-awselb"), []byte(skeletonRestartDetach), 0600); err != nil {
			return errors.WithStack(err)
		}

		if err = ioutil.WriteFile(filepath.Join(t.path, ".remote", "02_restart_module", "01_restart.bwcmd"), []byte(skeletonRestart), 0600); err != nil {
			return errors.WithStack(err)
		}

		if err = ioutil.WriteFile(filepath.Join(t.path, ".remote", "02_restart_module", "02_post_restart.attach-awselb"), []byte(skeletonRestartAttach), 0600); err != nil {
			return errors.WithStack(err)
		}

		if err = ioutil.WriteFile(filepath.Join(t.path, ".remote", "03_final.bwcmd"), []byte(skeletonFinal), 0600); err != nil {
			return errors.WithStack(err)
		}
	}

	return nil
}
