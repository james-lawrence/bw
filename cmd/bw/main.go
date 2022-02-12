package main

import (
	"context"
	"fmt"
	"log"
	"net"
	"os"
	"sync"
	"syscall"

	"github.com/james-lawrence/bw/agent"
	"github.com/james-lawrence/bw/cmd"
	"github.com/james-lawrence/bw/cmd/commandutils"
	"github.com/james-lawrence/bw/internal/x/debugx"
	"github.com/james-lawrence/bw/internal/x/systemx"

	"github.com/alecthomas/kingpin"
)

type global struct {
	systemIP  net.IP
	cluster   *clusterCmd
	ctx       context.Context
	shutdown  context.CancelFunc
	cleanup   *sync.WaitGroup
	verbosity int
}

func main() {
	var (
		err             error
		cleanup, cancel = context.WithCancel(context.Background())
		systemip        = systemx.HostIP(systemx.HostnameOrLocalhost())
		global          = &global{
			systemIP: systemip,
			cluster:  &clusterCmd{},
			ctx:      cleanup,
			shutdown: cancel,
			cleanup:  &sync.WaitGroup{},
		}

		agentcmd = &agentCmd{
			config: agent.NewConfig(agent.ConfigOptionDefaultBind(systemip)),
			global: global,
		}
		client = &deployCmd{
			global: global,
		}
		info = &agentInfo{
			global: global,
		}
		notify = &agentNotify{
			config: agent.NewConfig(agent.ConfigOptionDefaultBind(systemip)),
			global: global,
		}
		agentctl = &actlCmd{
			global: global,
		}
		me          = &me{global: global}
		workspace   = &workspaceCmd{global: global}
		environment = &environmentCmd{global: global}
	)

	log.SetFlags(log.Flags() | log.Lshortfile)
	go debugx.DumpOnSignal(cleanup, syscall.SIGUSR2)
	go systemx.Cleanup(global.ctx, global.shutdown, global.cleanup, os.Kill, os.Interrupt)(func() {
		log.Println("waiting for systems to shutdown")
	})

	app := kingpin.New("bearded-wookie", "deployment system").Version(cmd.Version)
	app.Command("version", "display the release version").Action(func(*kingpin.ParseContext) error {
		fmt.Println(cmd.Version)
		return nil
	})

	app.Flag("verbose", "increase verbosity of logging").Short('v').Default("0").Action(func(*kingpin.ParseContext) error {
		commandutils.LogEnv(global.verbosity)
		return nil
	}).CounterVar(&global.verbosity)

	me.configure(app.Command("me", "commands revolving around the users profile"))
	agentcmd.configure(app.Command("agent", "agent that manages deployments"))
	notify.configure(app.Command("notify", "watch for and emit deployment notifications"))
	client.configure(app.Command("deploy", "deploy to nodes within the cluster"))
	workspace.configure(app.Command("workspace", "workspace related commands"))
	environment.configure(app.Command("environment", "environment related commands"))
	info.configure(app.Command("info", "retrieve info about nodes within the cluster").Hidden())
	agentctl.configure(app.Command("agent-control", "shutdown agents on remote systems").Alias("actl").Hidden())

	if _, err = app.Parse(os.Args[1:]); err != nil {
		commandutils.LogCause(err)
		cancel()
	}

	global.cleanup.Wait()
}
