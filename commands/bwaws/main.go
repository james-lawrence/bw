package main

import (
	"context"
	"log"
	"net"
	"os"
	"path/filepath"
	"sync"
	"syscall"

	"github.com/james-lawrence/bw"
	"github.com/james-lawrence/bw/agent"
	"github.com/james-lawrence/bw/commands"
	"github.com/james-lawrence/bw/x/debugx"
	"github.com/james-lawrence/bw/x/systemx"

	"github.com/alecthomas/kingpin"
)

type global struct {
	systemIP net.IP
	ctx      context.Context
	shutdown context.CancelFunc
	cleanup  *sync.WaitGroup
}

// ./bin/bwaws dns --hzone=Z1QRT9PG2F57XP --region="us-east-1" --cluster=127.0.0.1:2001 --cluster=127.0.0.2:2001 --agent-config=".bwagent1/agent.config"
func main() {
	var (
		err             error
		exitCode        int
		cleanup, cancel = context.WithCancel(context.Background())
		systemip        = systemx.HostIP(systemx.HostnameOrLocalhost())
		global          = &global{
			systemIP: systemx.HostIP(systemx.HostnameOrLocalhost()),
			ctx:      cleanup,
			shutdown: cancel,
			cleanup:  &sync.WaitGroup{},
		}
		cdns = &cmdDNS{
			global:         global,
			config:         agent.NewConfig(agent.ConfigOptionDefaultBind(systemip)),
			configLocation: bw.DefaultLocation(filepath.Join(bw.DefaultEnvironmentName, bw.DefaultAgentConfig), ""),
		}
	)

	log.SetFlags(log.Flags() | log.Lshortfile)
	go debugx.DumpOnSignal(cleanup, syscall.SIGUSR2)
	go systemx.Cleanup(global.ctx, global.shutdown, global.cleanup, os.Kill, os.Interrupt)(func() {
		log.Println("waiting for systems to shutdown")
	})
	app := kingpin.New("bwaws", "bearded wookie utility programs for AWS").Version(commands.Version)
	cdns.Configure(app.Command("dns", "connects to the provided bearded wookie cluster"))

	if _, err = app.Parse(os.Args[1:]); err != nil {
		log.Printf("%v\n", err)
		cancel()
		exitCode = 1
	}

	global.cleanup.Wait()

	os.Exit(exitCode)
}
