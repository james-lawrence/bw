package debian

import (
	"context"
	"eg/compute/errorsx"
	"eg/compute/maintainer"
	"embed"
	"io/fs"
	"time"

	"github.com/egdaemon/eg/runtime/wasi/eg"
	"github.com/egdaemon/eg/runtime/wasi/egenv"
	"github.com/egdaemon/eg/runtime/wasi/eggit"
	"github.com/egdaemon/eg/runtime/wasi/shell"
	"github.com/egdaemon/eg/runtime/x/wasi/egdebuild"
)

//go:embed .debskel
var debskel embed.FS

func cachedir() string {
	return egenv.CacheDirectory(".dist", "bearded-wookie")
}

var (
	gcfg egdebuild.Config
)

func init() {
	c := eggit.EnvCommit()
	gcfg = egdebuild.New(
		"bearded-wookie",
		"",
		cachedir(),
		egdebuild.Option.Maintainer(maintainer.Name, maintainer.Email),
		egdebuild.Option.SigningKeyID(maintainer.GPGFingerprint),
		egdebuild.Option.ChangeLogDate(c.Committer.When),
		egdebuild.Option.Version("0.0.:autopatch:"),
		egdebuild.Option.Description("distributed configuration management", "bearded-wookie is a distributed configuration management system\n designed for high availability and minimal infrastructure overhead."),
		egdebuild.Option.Debian(errorsx.Must(fs.Sub(debskel, ".debskel"))),
		egdebuild.Option.DependsBuild("golang-1.24", "dh-make", "debhelper"),
	)
}

func Prepare(ctx context.Context, o eg.Op) error {
	debdir := cachedir()
	sruntime := shell.Runtime()
	return eg.Sequential(
		shell.Op(
			sruntime.Newf("rm -rf %s", debdir),
			sruntime.Newf("mkdir -p %s", debdir),
			sruntime.Newf("git clone --depth 1 file://${PWD}/ %s", debdir),
		),
		egdebuild.Prepare(Runner(), errorsx.Must(fs.Sub(debskel, ".debskel"))),
	)(ctx, o)
}

// container for this package.
func Runner() eg.ContainerRunner {
	return eg.Container("bw.debuild.ubuntu")
}

func Build(ctx context.Context, o eg.Op) error {
	return eg.Sequential(
		// useful for resolving build issues on ubuntu's workers
		egdebuild.Build(gcfg, egdebuild.Option.Distro("oracular"), egdebuild.Option.BuildBinary(time.Minute)),
		// shell.Op(shell.New("false")),
		eg.Parallel(
			egdebuild.Build(gcfg, egdebuild.Option.Distro("jammy")),
			egdebuild.Build(gcfg, egdebuild.Option.Distro("noble")),
			egdebuild.Build(gcfg, egdebuild.Option.Distro("oracular")),
			egdebuild.Build(gcfg, egdebuild.Option.Distro("plucky")),
		),
	)(ctx, o)
}

func Upload(ctx context.Context, o eg.Op) error {
	return egdebuild.UploadDPut(gcfg, errorsx.Must(fs.Sub(debskel, ".debskel")))(ctx, o)
}
