package deploy

import (
	"crypto/sha256"
	"io"
	"log"
	"os"
	"path/filepath"
	"time"

	"github.com/james-lawrence/bw"
	"github.com/james-lawrence/bw/agent"
	"github.com/james-lawrence/bw/archive"
	"github.com/james-lawrence/bw/cmd/commandutils"
	"github.com/james-lawrence/bw/internal/iox"
	"github.com/james-lawrence/bw/vcsinfo"
	"github.com/pkg/errors"
	"google.golang.org/protobuf/proto"
)

// Snapshot packs the deployspace into out, and writes the deployment metadata, the same data stored
// alongside the bootstrap archive, to metadata. metadata is optional.
func Snapshot(ctx Context, out io.WriteCloser, metadata io.Writer) (err error) {
	var (
		config    agent.ConfigClient
		commitish string
	)

	if config, err = commandutils.ReadConfiguration(ctx.Environment); err != nil {
		return errors.Wrap(err, "failed to load configuration")
	}

	log.Println("pid", os.Getpid())

	if err = os.WriteFile(filepath.Join(config.Deployment.DataDir, bw.EnvFile), []byte(config.Environment), 0600); err != nil {
		return errors.Wrap(err, "failed to crreate bw.env")
	}

	if commitish, err = commandutils.RunLocalDirectives(ctx.Context, config); err != nil {
		return errors.Wrap(err, "failed to run local directives")
	}

	if _, err := os.Stat(filepath.Join(config.Dir(), bw.AuthKeysFile)); !os.IsNotExist(err) {
		if err = iox.Copy(filepath.Join(config.Dir(), bw.AuthKeysFile), filepath.Join(config.Deployment.DataDir, bw.AuthKeysFile)); err != nil {
			return err
		}
	}

	defer out.Close()

	digest := sha256.New()
	if err = archive.Pack(io.MultiWriter(out, digest), config.Deployment.DataDir); err != nil {
		return err
	}

	if metadata == nil {
		return nil
	}

	sum := digest.Sum(nil)
	dopts := agent.DeployOptions{
		Timeout: int64(config.Deployment.Timeout),
	}

	raw, err := proto.Marshal(agent.DeployCommandDone(
		vcsinfo.CurrentUserDisplay(config.WorkDir()),
		(&agent.Archive{
			DeploymentID: sum,
			Checksum:     sum,
			Commit:       commitish,
			Ts:           time.Now().UnixNano(),
		}).DeployOption,
		dopts.DeployOption,
	))
	if err != nil {
		return errors.Wrap(err, "failed to encode deployment metadata")
	}

	if _, err = metadata.Write(raw); err != nil {
		return errors.Wrap(err, "failed to write deployment metadata")
	}

	return nil
}
