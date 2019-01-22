package commandutils

import (
	"io/ioutil"
	"os"
	"path/filepath"

	"github.com/james-lawrence/bw"
	"github.com/james-lawrence/bw/agent"
	"github.com/james-lawrence/bw/deployment"
	"github.com/james-lawrence/bw/directives/shell"
	"github.com/pkg/errors"
)

// determine if we need to run any remote tasks.
func RemoteTasksAvailable(config agent.ConfigClient) bool {
	_, err := os.Stat(filepath.Join(config.DeployDataDir, deployment.LocalDirName))
	return os.IsExist(err)
}

// RunLocalDirectives runs local directives, used to build archives prior to deploying.
func RunLocalDirectives(config agent.ConfigClient) (err error) {
	var (
		sctx    shell.Context
		dctx    deployment.DeployContext
		archive agent.Archive
	)

	if err = ioutil.WriteFile(filepath.Join(config.DeployDataDir, bw.EnvFile), []byte(config.Environment), 0600); err != nil {
		return err
	}

	local := NewClientPeer()

	if sctx, err = shell.DefaultContext(); err != nil {
		return err
	}

	archive = agent.Archive{}

	dopts := agent.DeployOptions{
		Timeout: int64(config.DeployTimeout),
	}

	opts := []deployment.DeployContextOption{
		deployment.DeployContextOptionLog(deployment.StdErrLogger("[LOCAL] ")),
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
