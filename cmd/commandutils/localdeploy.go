package commandutils

import (
	"context"
	"log"
	"os"
	"path/filepath"

	"github.com/james-lawrence/bw"
	"github.com/james-lawrence/bw/agent"
	"github.com/james-lawrence/bw/deployment"
	"github.com/james-lawrence/bw/directives/shell"
	"github.com/james-lawrence/bw/internal/debugx"
	"github.com/james-lawrence/bw/internal/errorsx"
	"github.com/james-lawrence/bw/vcsinfo"
	"github.com/pkg/errors"
)

// RemoteTasksAvailable determine if we need to run any remote tasks.
func RemoteTasksAvailable(config agent.ConfigClient) bool {
	debugx.Println("checking if remote tasks exist", filepath.Join(config.Deployment.DataDir, deployment.RemoteDirName))
	defer debugx.Println("done checking")

	_, err := os.Stat(filepath.Join(config.Deployspace(), deployment.RemoteDirName))
	errorsx.Log(errors.Wrap(err, "stat failed"))
	return err == nil
}

// RunLocalDirectives runs local directives, used to build archives prior to deploying.
func RunLocalDirectives(ctx context.Context, config agent.ConfigClient) (commitish string, err error) {
	var (
		sctx    shell.Context
		dctx    *deployment.DeployContext
		environ []string
		cdir    string = config.Deployspace()
		root           = config.WorkDir()
	)

	displayname := vcsinfo.CurrentUserDisplay(config.WorkDir())
	commitish = vcsinfo.Commitish(config.WorkDir(), config.Deployment.CommitRef)
	log.Println("vcs.commit", config.Deployment.CommitRef, "->", commitish)

	if err = os.WriteFile(filepath.Join(cdir, bw.EnvFile), []byte(config.Environment), 0600); err != nil {
		return commitish, err
	}

	if environ, err = shell.EnvironFromFile(filepath.Join(cdir, bw.EnvFile)); err != nil {
		return commitish, err
	}

	local := NewClientPeer()

	if sctx, err = shell.DefaultContext(); err != nil {
		return commitish, err
	}

	sctx = shell.NewContext(
		sctx,
		shell.OptionEnviron(append(environ, sctx.Environ...)),
		shell.OptionDir(root),
		shell.OptionWorkDir(root),
		shell.OptionBWConfigDir(config.Dir()),
		shell.OptionVCSCommit(commitish),
	)

	dctx, err = deployment.NewDeployContext(
		ctx,
		cdir,
		local,
		displayname,
		&agent.DeployOptions{
			Timeout: int64(config.Deployment.Timeout),
		},
		&agent.Archive{
			Commit: commitish,
		},
		deployment.DeployContextOptionLog(deployment.StdErrLogger("[LOCAL] ")),
		deployment.DeployContextOptionTempRoot(config.Dir()),
		deployment.DeployContextOptionCacheRoot(config.Dir()),
		deployment.DeployContextOptionDisableReset,
	)

	if err != nil {
		return commitish, errors.Wrap(err, "failed to create deployment context")
	}

	deploy := deployment.NewDirective(
		deployment.DirectiveOptionShellContext(sctx),
		deployment.DirectiveOptionDir(deployment.LocalDirName),
	)
	deploy.Deploy(dctx)

	return commitish, deployment.AwaitDeployResult(dctx).Error
}
