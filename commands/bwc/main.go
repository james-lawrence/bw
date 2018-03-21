package main

import (
	"context"
	"log"
	"net"
	"os"
	"sync"
	"syscall"

	"github.com/james-lawrence/bw/commands"
	"github.com/james-lawrence/bw/x/debugx"
	"github.com/james-lawrence/bw/x/systemx"

	"github.com/alecthomas/kingpin"
)

type global struct {
	systemIP net.IP
	// cluster  *commandutils.ClusterCmd
	ctx      context.Context
	shutdown context.CancelFunc
	cleanup  *sync.WaitGroup
}

func main() {
	var (
		err             error
		cleanup, cancel = context.WithCancel(context.Background())
		// systemip        = systemx.HostIP(systemx.HostnameOrLocalhost())
		global = &global{
			systemIP: systemx.HostIP(systemx.HostnameOrLocalhost()),
			// cluster:  &clusterCmd{},
			ctx:      cleanup,
			shutdown: cancel,
			cleanup:  &sync.WaitGroup{},
		}

		client = &deployCmd{
			global: global,
		}
		info = &agentInfo{
			global: global,
		}
		agentctl = &actlCmd{
			global: global,
		}
		workspace   = &workspaceCmd{global: global}
		environment = &environmentCmd{global: global}
	)

	log.SetFlags(log.Flags() | log.Lshortfile)
	go debugx.DumpOnSignal(cleanup, syscall.SIGUSR2)
	go systemx.Cleanup(global.ctx, global.shutdown, global.cleanup, os.Kill, os.Interrupt)(func() {
		log.Println("waiting for systems to shutdown")
	})
	app := kingpin.New("bearded-wookie", "deployment system").Version(commands.Version)
	client.configure(app.Command("deploy", "deploy to nodes within the cluster"))
	workspace.configure(app.Command("workspace", "workspace related commands"))
	environment.configure(app.Command("environment", "environment related commands"))
	info.configure(app.Command("info", "retrieve info about nodes within the cluster").Hidden())
	agentctl.configure(app.Command("agent-control", "shutdown agents on remote systems").Alias("actl").Hidden())

	if _, err = app.Parse(os.Args[1:]); err != nil {
		log.Printf("%+v\n", err)
		cancel()
	}

	global.cleanup.Wait()
}
