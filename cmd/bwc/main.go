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
	"github.com/james-lawrence/bw"
	"github.com/james-lawrence/bw/agent"
	"github.com/james-lawrence/bw/cmd/autocomplete"
	"github.com/james-lawrence/bw/cmd/bwc/agentcmd"
	"github.com/james-lawrence/bw/cmd/bwc/cmdopts"
	"github.com/james-lawrence/bw/cmd/commandutils"
	"github.com/james-lawrence/bw/internal/x/debugx"
	"github.com/james-lawrence/bw/internal/x/systemx"
	"github.com/posener/complete"
	"github.com/willabides/kongplete"
)

func main() {
	var shellCli struct {
		cmdopts.Global
		Environment        cmdEnv                       `cmd:"" help:"nvironment related commands"`
		Deploy             cmdDeploy                    `cmd:"" help:"deployment related commands"`
		Me                 cmdMe                        `cmd:"" help:"commands for managing the user's profile"`
		Info               cmdInfo                      `cmd:"" help:"retrieve information from an environment" hidden:""`
		Workspace          cmdWorkspace                 `cmd:"" help:"workspace related commands"`
		InstallCompletions kongplete.InstallCompletions `cmd:"" help:"install shell completions"`
		Agent              agentcmd.CmdDaemon           `cmd:"" help:"agent that manages deployments"`
		AgentControl       agentcmd.CmdControl          `cmd:"" name:"actl" help:"remote administration of the environment" aliases:"agent-control"`
		Notify             agentcmd.Notify              `cmd:"" help:"watch for and emit deployment notifications"`
	}

	var (
		err                 error
		ctx                 *kong.Context
		systemip            = systemx.HostIP(systemx.HostnameOrLocalhost())
		agentconfigdefaults = agent.NewConfig(agent.ConfigOptionDefaultBind(systemip))
		peeringopts         = cmdopts.Peering{}
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
		kong.Vars{
			"vars_bw_default_env_name":                     bw.DefaultEnvironmentName,
			"vars_bw_default_deployspace_directory":        bw.DefaultDeployspaceDir,
			"vars_bw_default_deployspace_config_directory": bw.DefaultDeployspaceConfigDir,
			"vars_bw_default_agent_configuration_location": bw.DefaultLocation(filepath.Join(bw.DefaultEnvironmentName, bw.DefaultAgentConfig), ""),
			"vars_bw_default_agent_address":                agentconfigdefaults.P2PBind.String(),
			"env_bw_agent_bind_primary":                    bw.EnvAgentP2PBind,
			"env_bw_agent_bind_advertised":                 bw.EnvAgentP2PAdvertised,
			"env_bw_agent_bind_secondary":                  bw.EnvAgentP2PAlternatesBind,
		},
		kong.UsageOnError(),
		kong.Bind(&shellCli.Global),
		kong.Bind(&agentconfigdefaults),
		kong.Bind(&peeringopts),
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