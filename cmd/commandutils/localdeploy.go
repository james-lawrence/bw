package commandutils

import (
	"context"
	"os"
	"path/filepath"

	"github.com/james-lawrence/bw"
	"github.com/james-lawrence/bw/agent"
	"github.com/james-lawrence/bw/deployment"
	"github.com/james-lawrence/bw/directives/shell"
	"github.com/james-lawrence/bw/internal/debugx"
	"github.com/james-lawrence/bw/internal/logx"
	"github.com/pkg/errors"
)

// RemoteTasksAvailable determine if we need to run any remote tasks.
func RemoteTasksAvailable(config agent.ConfigClient) bool {
	debugx.Println("checking if remote tasks exist", filepath.Join(config.Deployment.DataDir, deployment.RemoteDirName))
	defer debugx.Println("done checking")

	_, err := os.Stat(filepath.Join(config.Deployspace(), deployment.RemoteDirName))
	logx.MaybeLog(errors.Wrap(err, "stat failed"))
	return err == nil
}

// RunLocalDirectives runs local directives, used to build archives prior to deploying.
func RunLocalDirectives(config agent.ConfigClient) (err error) {
	var (
		sctx    shell.Context
		dctx    *deployment.DeployContext
		environ []string
		cdir    string = config.Deployspace()
		root           = config.WorkDir()
	)

	if err = os.WriteFile(filepath.Join(cdir, bw.EnvFile), []byte(config.Environment), 0600); err != nil {
		return err
	}

	if environ, err = shell.EnvironFromFile(filepath.Join(cdir, bw.EnvFile)); err != nil {
		dctx.Done(err)
		return
	}

	local := NewClientPeer()

	if sctx, err = shell.DefaultContext(); err != nil {
		return err
	}

	sctx = shell.NewContext(
		sctx,
		shell.OptionEnviron(append(environ, sctx.Environ...)),
		shell.OptionDir(root),
	)

	dctx, err = deployment.NewDeployContext(
		context.Background(),
		cdir,
		local,
		bw.DisplayName(),
		&agent.DeployOptions{
			Timeout: int64(config.Deployment.Timeout),
		},
		&agent.Archive{},
		deployment.DeployContextOptionLog(deployment.StdErrLogger("[LOCAL] ")),
		deployment.DeployContextOptionTempRoot(config.Dir()),
		deployment.DeployContextOptionCacheRoot(config.Dir()),
		deployment.DeployContextOptionDisableReset,
	)
	if err != nil {
		return errors.Wrap(err, "failed to create deployment context")
	}

	deploy := deployment.NewDirective(
		deployment.DirectiveOptionShellContext(sctx),
		deployment.DirectiveOptionDir(deployment.LocalDirName),
	)
	deploy.Deploy(dctx)

	return deployment.AwaitDeployResult(dctx).Error
}
