package deploy

import (
	"io"
	"log"
	"os"
	"path/filepath"

	"github.com/james-lawrence/bw"
	"github.com/james-lawrence/bw/agent"
	"github.com/james-lawrence/bw/archive"
	"github.com/james-lawrence/bw/cmd/commandutils"
	"github.com/james-lawrence/bw/deployment"
	"github.com/james-lawrence/bw/directives/shell"
	"github.com/james-lawrence/bw/internal/errorsx"
	"github.com/james-lawrence/bw/internal/iox"
	"github.com/logrusorgru/aurora"
	"github.com/pkg/errors"
)

func Locally(ctx *Context, debug bool) (err error) {
	var (
		dst       *os.File
		sctx      shell.Context
		dctx      *deployment.DeployContext
		root      string
		config    agent.ConfigClient
		commitish string
	)

	if config, err = commandutils.ReadConfiguration(ctx.Environment); err != nil {
		return errors.Wrap(err, "unable to read configuration")
	}

	if err = os.WriteFile(filepath.Join(config.WorkDir(), config.Deployment.DataDir, bw.EnvFile), []byte(config.Environment), 0600); err != nil {
		return errors.WithStack(err)
	}

	if _, err := os.Stat(filepath.Join(config.Dir(), bw.AuthKeysFile)); !os.IsNotExist(err) {
		if err = iox.Copy(filepath.Join(config.Dir(), bw.AuthKeysFile), filepath.Join(config.Deployment.DataDir, bw.AuthKeysFile)); err != nil {
			return errors.WithStack(err)
		}
	}

	if commitish, err = commandutils.RunLocalDirectives(ctx.Context, config); err != nil {
		return errors.Wrap(err, "failed to run local directives")
	}

	local := commandutils.NewClientPeer()

	if sctx, err = shell.DefaultContext(); err != nil {
		return err
	}

	if root, err = os.MkdirTemp("", "bw-local-deploy-*"); err != nil {
		return err
	}

	if debug {
		log.Printf("building directory '%s' will remain after exit\n", root)
		defer func() {
			err = errorsx.Compact(err, errorsx.Notification(errors.Errorf("%s build directory '%s' being left on disk", aurora.NewAurora(true).Yellow("WARN"), root)))
		}()
	} else {
		defer os.RemoveAll(root)
	}

	if dst, err = os.CreateTemp("", "bwarchive"); err != nil {
		return errors.Wrap(err, "archive creation failed")
	}

	defer os.Remove(dst.Name())
	defer dst.Close()

	if err = archive.Pack(dst, filepath.Join(config.WorkDir(), config.Deployment.DataDir)); err != nil {
		return errors.Wrap(err, "failed to pack archive")
	}

	if _, err = dst.Seek(0, io.SeekStart); err != nil {
		return errors.WithStack(err)
	}

	if err = os.MkdirAll(filepath.Join(root, bw.DirArchive), 0700); err != nil {
		return errors.Wrap(err, "failed to create archive directory")
	}

	if err = archive.Unpack(filepath.Join(root, bw.DirArchive), dst); err != nil {
		return errors.Wrap(err, "failed to unpack archive")
	}

	dctx, err = deployment.NewRemoteDeployContext(
		ctx.Context,
		root,
		local,
		bw.DisplayName(),
		&agent.DeployOptions{
			Timeout:   int64(config.Deployment.Timeout),
			Heartbeat: int64(ctx.Heartbeat),
		},
		&agent.Archive{
			Location: dst.Name(),
			Commit:   commitish,
		},
		deployment.DeployContextOptionCacheRoot(config.Dir()),
		deployment.DeployContextOptionDisableReset,
	)
	if err != nil {
		return errors.Wrap(err, "failed to create deployment context")
	}

	deploy := deployment.NewDirective(
		deployment.DirectiveOptionShellContext(sctx),
	)
	deploy.Deploy(dctx)

	result := deployment.AwaitDeployResult(dctx)

	return result.Error
}
