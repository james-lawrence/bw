// Package bwc is the user client which focuses on human friendly behaviors not system administration, and not on backwards compatibility.
package main

import (
	"context"
	"log"
	"os"
	"path/filepath"
	"sync"
	"syscall"

	"github.com/alecthomas/kong"
	"github.com/egdaemon/gdx"
	"github.com/egdaemon/gdx/konggdx"
	"github.com/james-lawrence/bw/agent"
	"github.com/james-lawrence/bw/cmd/autocomplete"
	"github.com/james-lawrence/bw/cmd/bw/agentcmd"
	"github.com/james-lawrence/bw/cmd/bw/cmdopts"
	"github.com/james-lawrence/bw/cmd/commandutils"
	"github.com/james-lawrence/bw/internal/contextx"
	"github.com/james-lawrence/bw/internal/debugx"
	"github.com/james-lawrence/bw/internal/systemx"
	"github.com/james-lawrence/bw/internal/userx"
	"github.com/posener/complete"
	"github.com/willabides/kongplete"
)

// cli is the command line of bw.
type cli struct {
	cmdopts.Global
	Version            cmdVersion                   `cmd:"" help:"display versioning information"`
	Environment        cmdEnv                       `cmd:"" help:"environment related commands"`
	Deploy             cmdDeploy                    `cmd:"" help:"deployment related commands"`
	Redeploy           cmdDeployRedeploy            `cmd:"" name:"redeploy" help:"redeploy an archive to nodes within the cluster of the specified environment"`
	Me                 cmdMe                        `cmd:"" help:"commands for managing the user's profile"`
	Notary             cmdNotary                    `cmd:"" help:"retrieve and manage permissions"`
	Workspace          cmdWorkspace                 `cmd:"" help:"workspace related commands"`
	Agent              agentcmd.CmdDaemon           `cmd:"" help:"agent that manages deployments"`
	AgentControl       agentcmd.CmdControl          `cmd:"" name:"actl" help:"remote administration of the environment" aliases:"agent-control"`
	Notify             agentcmd.Notify              `cmd:"" help:"watch for and emit deployment notifications"`
	GDX                konggdx.Commands             `cmd:"" name:"gdx" help:"pull profiles/traces from a running bw agent debug socket"`
	InstallCompletions kongplete.InstallCompletions `cmd:"" help:"install shell completions"`
}

// gdxSocket is the unix socket the agent serves its debug surface on. it lives in the runtime directory, which
// systemd makes unique per agent instance (RuntimeDirectory=%p/%i), so agents sharing a host don't share a socket.
func gdxSocket() string {
	return filepath.Join(userx.DefaultRuntimeDirectory(), gdx.DefaultSocket)
}

// newcmdroot builds the parser used by main. defaults is bound so the command hooks (agentcmd.Config.AfterApply)
// can apply the flags and environment onto it. options are appended last, tests use them to override the
// kong.Vars or exit behavior without touching the real filesystem.
func newcmdroot(c *cli, defaults *agent.Config, options ...kong.Option) *kong.Kong {

	return kong.Must(
		c,
		append([]kong.Option{
			kong.Name("bw"),
			kong.Description("user frontend to bearded-wookie"),
			cmdopts.Vars(*defaults),
			kong.Vars{
				"vars_gdx_socket":         gdxSocket(),
				"vars_gdx_default_output": "-",
			},
			kong.UsageOnError(),
			kong.Bind(&c.Global),
			kong.Bind(defaults),
			kong.BindTo(c.Context, (*context.Context)(nil)), // konggdx commands take a plain context.
			cmdopts.Mappers(),
		}, options...)...,
	)
}

func main() {
	var shellCli cli

	var (
		err                 error
		ctx                 *kong.Context
		systemip            = systemx.HostIP(systemx.HostnameOrLocalhost())
		agentconfigdefaults = agent.NewConfig(agent.ConfigOptionDefaultBind(systemip))
	)

	shellCli.Cleanup = &sync.WaitGroup{}
	shellCli.Context = contextx.WithWaitGroup(context.Background(), shellCli.Cleanup)
	shellCli.Context, shellCli.Shutdown = context.WithCancel(shellCli.Context)

	log.SetFlags(log.Flags() | log.Lshortfile)
	go debugx.DumpOnSignal(shellCli.Context, syscall.SIGUSR2)
	go systemx.Cleanup(shellCli.Context, shellCli.Shutdown, shellCli.Cleanup, os.Kill, os.Interrupt)(func() {
		log.Println("waiting for systems to shutdown")
	})

	parser := newcmdroot(&shellCli, &agentconfigdefaults)

	// Run kongplete.Complete to handle completion requests
	kongplete.Complete(parser,
		kongplete.WithPredictor("bw.environment", complete.PredictFunc(autocomplete.Deployspaces)),
	)

	if ctx, err = parser.Parse(os.Args[1:]); err != nil {
		_ = commandutils.LogCause(err)
		os.Exit(1)
	}

	// only the long running agents serve the debug surface. serving it removes any existing socket, short lived
	// commands must not take the socket of a running agent.
	switch ctx.Command() {
	case "agent", "agent runtime", "agent coordinator":
		go func() {
			if err := (konggdx.Serve{Socket: gdxSocket()}).Run(shellCli.Context); err != nil {
				log.Println("gdx debug server failed", err)
			}
		}()
	}

	if err = commandutils.LogCause(ctx.Run()); err != nil {
		shellCli.Shutdown()
	}

	shellCli.Cleanup.Wait()
	if err != nil {
		os.Exit(1)
	}
}
