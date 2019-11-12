package commandutils

import (
	"io/ioutil"
	"os"
	"path/filepath"

	"github.com/james-lawrence/bw"
	"github.com/james-lawrence/bw/agent"
	"github.com/james-lawrence/bw/deployment"
	"github.com/james-lawrence/bw/directives/shell"
	"github.com/james-lawrence/bw/internal/x/debugx"
	"github.com/james-lawrence/bw/internal/x/logx"
	"github.com/pkg/errors"
)

// RemoteTasksAvailable determine if we need to run any remote tasks.
func RemoteTasksAvailable(config agent.ConfigClient) bool {
	debugx.Println("checking if remote tasks exist", filepath.Join(config.DeployDataDir, deployment.RemoteDirName))
	defer debugx.Println("done checking")
	_, err := os.Stat(filepath.Join(config.DeployDataDir, deployment.RemoteDirName))
	logx.MaybeLog(errors.Wrap(err, "stat failed"))
	return os.IsExist(err) || err == nil
}

// RunLocalDirectives runs local directives, used to build archives prior to deploying.
func RunLocalDirectives(config agent.ConfigClient) (err error) {
	var (
		sctx    shell.Context
		dctx    deployment.DeployContext
		archive agent.Archive
		environ []string
	)

	if err = ioutil.WriteFile(filepath.Join(config.DeployDataDir, bw.EnvFile), []byte(config.Environment), 0600); err != nil {
		return err
	}

	if environ, err = shell.EnvironFromFile(filepath.Join(config.DeployDataDir, bw.EnvFile)); err != nil {
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
		shell.OptionDir(config.DeployDataDir),
	)

	archive = agent.Archive{}

	dopts := agent.DeployOptions{
		Timeout: int64(config.DeployTimeout),
	}

	opts := []deployment.DeployContextOption{
		deployment.DeployContextOptionLog(deployment.StdErrLogger("[LOCAL] ")),
		deployment.DeployContextOptionDisableReset,
	}

	if dctx, err = deployment.NewDeployContext(config.DeployDataDir, local, dopts, archive, opts...); err != nil {
		return errors.Wrap(err, "failed to create deployment context")
	}

	deploy := deployment.NewDirective(
		deployment.DirectiveOptionShellContext(sctx),
		deployment.DirectiveOptionDir(deployment.LocalDirName),
	)
	deploy.Deploy(dctx)

	return deployment.AwaitDeployResult(dctx).Error
}
