package deploy

import (
	"io"
	"io/ioutil"
	"log"
	"os"
	"path/filepath"

	"github.com/james-lawrence/bw"
	"github.com/james-lawrence/bw/agent"
	"github.com/james-lawrence/bw/archive"
	"github.com/james-lawrence/bw/cmd/commandutils"
	"github.com/james-lawrence/bw/internal/x/iox"
	"github.com/pkg/errors"
)

func Snapshot(ctx Context, out io.WriteCloser) (err error) {
	var (
		config agent.ConfigClient
	)

	if config, err = commandutils.ReadConfiguration(ctx.Environment); err != nil {
		return errors.Wrap(err, "failed to load configuration")
	}

	log.Println("pid", os.Getpid())

	if err = ioutil.WriteFile(filepath.Join(config.DeployDataDir, bw.EnvFile), []byte(config.Environment), 0600); err != nil {
		return errors.Wrap(err, "failed to crreate bw.env")
	}

	if err = commandutils.RunLocalDirectives(config); err != nil {
		return errors.Wrap(err, "failed to run local directives")
	}

	if _, err := os.Stat(filepath.Join(config.Dir(), bw.AuthKeysFile)); !os.IsNotExist(err) {
		if err = iox.Copy(filepath.Join(config.Dir(), bw.AuthKeysFile), filepath.Join(config.DeployDataDir, bw.AuthKeysFile)); err != nil {
			return err
		}
	}

	defer out.Close()

	if err = archive.Pack(out, config.DeployDataDir); err != nil {
		return err
	}

	return nil
}