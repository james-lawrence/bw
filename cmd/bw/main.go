// Package bwc is the user client which focuses on human friendly behaviors not system administration, and not on backwards compatibility.
package main

import (
	"context"
	"log"
	"net"
	"os"
	"path/filepath"
	"reflect"
	"sync"
	"syscall"

	"github.com/alecthomas/kong"
	"github.com/james-lawrence/bw"
	"github.com/james-lawrence/bw/agent"
	"github.com/james-lawrence/bw/cmd/autocomplete"
	"github.com/james-lawrence/bw/cmd/bw/agentcmd"
	"github.com/james-lawrence/bw/cmd/bw/cmdopts"
	"github.com/james-lawrence/bw/cmd/commandutils"
	"github.com/james-lawrence/bw/internal/x/debugx"
	"github.com/james-lawrence/bw/internal/x/systemx"
	"github.com/posener/complete"
	"github.com/willabides/kongplete"
)

func main() {
	var shellCli struct {
		cmdopts.Global
		Version            cmdVersion                   `cmd:"" help:"display versioning information"`
		Environment        cmdEnv                       `cmd:"" help:"environment related commands"`
		Deploy             cmdDeploy                    `cmd:"" help:"deployment related commands"`
		Redeploy           cmdDeployRedeploy            `cmd:"" name:"redeploy" help:"redeploy an archive to nodes within the cluster of the specified environment"`
		Me                 cmdMe                        `cmd:"" help:"commands for managing the user's profile"`
		Info               cmdInfo                      `cmd:"" help:"retrieve information from an environment"`
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
		kong.Name("bw"),
		kong.Description("user frontend to bearded-wookie"),
		kong.Vars{
			"vars_bw_default_env_name":                         bw.DefaultEnvironmentName,
			"vars_bw_default_deployspace_directory":            bw.DefaultDeployspaceDir,
			"vars_bw_default_deployspace_config_directory":     bw.DefaultDeployspaceConfigDir,
			"vars_bw_default_agent_configuration_location":     bw.DefaultLocation(filepath.Join(bw.DefaultEnvironmentName, bw.DefaultAgentConfig), ""),
			"vars_bw_default_agent_address":                    agentconfigdefaults.P2PBind.String(),
			"env_bw_agent_bind_primary":                        bw.EnvAgentP2PBind,
			"env_bw_agent_bind_advertised":                     bw.EnvAgentP2PAdvertised,
			"env_bw_agent_bind_secondary":                      bw.EnvAgentP2PAlternatesBind,
			"env_bw_agent_bootstrap_static":                    bw.EnvAgentClusterBootstrap,
			"env_bw_agent_bootstrap_dns_enabled":               bw.EnvAgentClusterEnableDNS,
			"env_bw_agent_bootstrap_aws_autoscaling_enabled":   bw.EnvAgentClusterEnableAWSAutoscaling,
			"env_bw_agent_bootstrap_gcloud_taget_pool_enabled": bw.EnvAgentClusterEnableGoogleCloudPool,
		},
		kong.UsageOnError(),
		kong.Bind(&shellCli.Global),
		kong.Bind(&agentconfigdefaults),
		kong.TypeMapper(reflect.TypeOf(&net.IP{}), kong.MapperFunc(cmdopts.ParseIP)),
		kong.TypeMapper(reflect.TypeOf(&net.TCPAddr{}), kong.MapperFunc(cmdopts.ParseTCPAddr)),
		kong.TypeMapper(reflect.TypeOf([]*net.TCPAddr(nil)), kong.MapperFunc(cmdopts.ParseTCPAddrArray)),
	)

	// Run kongplete.Complete to handle completion requests
	kongplete.Complete(parser,
		kongplete.WithPredictor("bw.environment", complete.PredictFunc(autocomplete.Deployspaces)),
	)

	if ctx, err = parser.Parse(os.Args[1:]); err != nil {
		commandutils.LogCause(err)
		os.Exit(1)
	}

	if err = commandutils.LogCause(ctx.Run()); err != nil {
		shellCli.Shutdown()
	}

	shellCli.Cleanup.Wait()
	if err != nil {
		os.Exit(1)
	}
	// ctx.FatalIfErrorf(err)
}
