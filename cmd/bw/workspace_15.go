//go:build !go1.16
// +build !go1.16

package main

import (
	"github.com/james-lawrence/bw/cmd/bw/cmdopts"
	"github.com/pkg/errors"
	"io/ioutil"
	"os"
	"path/filepath"
)

func (t *cmdWorkspaceCreate) Run(ctx *cmdopts.Global) (err error) {
	// TODO: tarball url instead of inlining like so.
	// - command: "echo %H %m %d %f %u %U %h %bwroot %bwcwd %%"
	// - command: git fetch --all
	// - command: echo "deploying ${DEPLOY_BRANCH}"
	// - command: git rev-parse --verify ${DEPLOY_BRANCH} > git.commit
	// - command: cd %bwcwd; git archive --format=tar.gz -o %bwroot/archive.tar.gz ${DEPLOY_BRANCH}
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

	if err = errors.WithStack(os.MkdirAll(t.Directory, 0755)); err != nil {
		return err
	}

	if err = errors.WithStack(os.MkdirAll(filepath.Join(t.Directory, ".remote"), 0755)); err != nil {
		return err
	}

	if t.Example {
		if err = os.WriteFile(filepath.Join(t.Directory, ".remote", "01_shell.bwcmd"), []byte(skeletonShellDirective), 0600); err != nil {
			return errors.WithStack(err)
		}

		if err = errors.WithStack(os.MkdirAll(filepath.Join(t.Directory, ".remote", "02_restart_module"), 0755)); err != nil {
			return err
		}

		if err = os.WriteFile(filepath.Join(t.Directory, ".remote", "02_restart_module", "00_pre_restart.detach-awselb"), []byte(skeletonRestartDetach), 0600); err != nil {
			return errors.WithStack(err)
		}

		if err = os.WriteFile(filepath.Join(t.Directory, ".remote", "02_restart_module", "01_restart.bwcmd"), []byte(skeletonRestart), 0600); err != nil {
			return errors.WithStack(err)
		}

		if err = os.WriteFile(filepath.Join(t.Directory, ".remote", "02_restart_module", "02_post_restart.attach-awselb"), []byte(skeletonRestartAttach), 0600); err != nil {
			return errors.WithStack(err)
		}

		if err = os.WriteFile(filepath.Join(t.Directory, ".remote", "03_final.bwcmd"), []byte(skeletonFinal), 0600); err != nil {
			return errors.WithStack(err)
		}
	}

	return nil
}
