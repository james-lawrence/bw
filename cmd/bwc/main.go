// Package bwc is the user client which focuses on human friendly behaviors not system administration, and not on backwards compatibility.
package main

import (
	"context"
	"log"
	"os"
	"sync"
	"syscall"

	"github.com/alecthomas/kong"
	"github.com/james-lawrence/bw"
	"github.com/james-lawrence/bw/cmd/autocomplete"
	"github.com/james-lawrence/bw/cmd/commandutils"
	"github.com/james-lawrence/bw/internal/x/debugx"
	"github.com/james-lawrence/bw/internal/x/systemx"
	"github.com/posener/complete"
	"github.com/willabides/kongplete"
)

type Global struct {
	Verbosity int                `help:"increase verbosity of logging" short:"v" type:"counter"`
	Context   context.Context    `kong:"-"`
	Shutdown  context.CancelFunc `kong:"-"`
	Cleanup   *sync.WaitGroup    `kong:"-"`
}

func (t Global) BeforeApply() error {
	commandutils.LogEnv(t.Verbosity)
	return nil
}

type BeardedWookieEnv struct {
	Environment string `arg:"" name:"environment" predictor:"bw.environment" default:"${vars.bw.default.env.name}"`
}

type BeardedWookieEnvRequired struct {
	Environment string `arg:"" name:"environment" predictor:"bw.environment"`
}

func main() {
	var shellCli struct {
		Global
		Deploy             deployCmd                    `cmd:"" help:"deployment related commands"`
		Me                 cmdMe                        `cmd:"" help:"commands for managing the user's profile"`
		Info               cmdInfo                      `cmd:"" help:"retrieve information from an environment" hidden:""`
		Workspace          cmdWorkspace                 `cmd:"" help:"workspace related commands"`
		InstallCompletions kongplete.InstallCompletions `cmd:"" help:"install shell completions"`
	}

	var (
		err error
		ctx *kong.Context
		// systemip = systemx.HostIP(systemx.HostnameOrLocalhost())
	)

	shellCli.Context, shellCli.Shutdown = context.WithCancel(context.Background())
	shellCli.Cleanup = &sync.WaitGroup{}

	log.SetFlags(log.Flags() | log.Lshortfile)
	go debugx.DumpOnSignal(shellCli.Context, syscall.SIGUSR2)
	go systemx.Cleanup(shellCli.Context, shellCli.Shutdown, shellCli.Cleanup, os.Kill, os.Interrupt)(func() {
		log.Println("waiting for systems to shutdown")
	})

	parser := kong.Must(
		&shellCli,
		kong.Name("bwc"),
		kong.Description("user frontend to bearded-wookie"),
		kong.UsageOnError(),
		kong.Bind(&shellCli.Global),
		kong.Vars{
			"vars.bw.default.env.name":              bw.DefaultEnvironmentName,
			"vars.bw.default.deployspace.directory": bw.DefaultDeployspaceDir,
		},
	)

	// Run kongplete.Complete to handle completion requests
	kongplete.Complete(parser,
		kongplete.WithPredictor("bw.environment", complete.PredictFunc(autocomplete.Deployspaces)),
		kongplete.WithPredictor("file", complete.PredictFiles("*")),
	)

	if ctx, err = parser.Parse(os.Args[1:]); err != nil {
		log.Fatalln(err)
	}

	ctx.FatalIfErrorf(commandutils.LogCause(ctx.Run()))
}
