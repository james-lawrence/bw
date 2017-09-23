package main

import (
	"context"
	"log"
	"net"
	"os"
	"sync"
	"syscall"

	"bitbucket.org/jatone/bearded-wookie"
	"bitbucket.org/jatone/bearded-wookie/agent"
	"bitbucket.org/jatone/bearded-wookie/commands"
	"bitbucket.org/jatone/bearded-wookie/x/debugx"
	"bitbucket.org/jatone/bearded-wookie/x/netx"
	"bitbucket.org/jatone/bearded-wookie/x/systemx"

	"github.com/alecthomas/kingpin"
)

type global struct {
	systemIP net.IP
	cluster  *cluster
	ctx      context.Context
	shutdown context.CancelFunc
	cleanup  *sync.WaitGroup
}

// agent: NETWORK=127.0.0.1; ./bin/bearded-wookie agent --agent-bind=$NETWORK:2000 --cluster-bind=$NETWORK:2001 --cluster-bind-raft=$NETWORK:2002 --cluster=127.0.0.3:2001 --cluster-minimum-required-peers=0 --cluster-maximum-join-attempts=10 --agent-config=".bwagent1/agent.config"
// agent: NETWORK=127.0.0.2; ./bin/bearded-wookie agent --agent-bind=$NETWORK:2000 --cluster-bind=$NETWORK:2001 --cluster-bind-raft=$NETWORK:2002 --cluster=127.0.0.2:2001 --cluster-minimum-required-peers=0 --cluster-maximum-join-attempts=10 --agent-config=".bwagent2/agent.config"
// agent: NETWORK=127.0.0.3; ./bin/bearded-wookie agent --agent-bind=$NETWORK:2000 --cluster-bind=$NETWORK:2001 --cluster-bind-raft=$NETWORK:2002 --cluster=127.0.0.2:2001 --cluster-minimum-required-peers=0 --cluster-maximum-join-attempts=10 --agent-config=".bwagent3/agent.config"
// agent: NETWORK=127.0.0.4; ./bin/bearded-wookie agent --agent-bind=$NETWORK:2000 --cluster-bind=$NETWORK:2001 --cluster-bind-raft=$NETWORK:2002 --cluster=127.0.0.2:2001 --cluster-minimum-required-peers=0 --cluster-maximum-join-attempts=10 --agent-config=".bwagent4/agent.config"
// client: NETWORK=127.0.0.6; ./bin/bearded-wookie deploy --cluster-bind=$NETWORK:2001

func main() {
	var (
		err             error
		cleanup, cancel = context.WithCancel(context.Background())
		systemip        = systemx.HostIP(systemx.HostnameOrLocalhost())
		global          = &global{
			systemIP: systemx.HostIP(systemx.HostnameOrLocalhost()),
			cluster:  &cluster{},
			ctx:      cleanup,
			shutdown: cancel,
			cleanup:  &sync.WaitGroup{},
		}
		clientConfig = agent.NewConfigClient()
		agentcmd     = &agentCmd{
			config: agent.NewConfig(),
			global: global,
			network: &net.TCPAddr{
				IP:   systemip,
				Port: 2000,
			},
			listener: netx.NewNoopListener(),
		}
		client = &deployCmd{
			config: clientConfig,
			global: global,
		}
		info = &agentInfo{
			node:   bw.MustGenerateID().String(),
			config: clientConfig,
			global: global,
		}
		envinit = &initCmd{
			global: global,
		}
	)

	log.SetFlags(log.Flags() | log.Lshortfile)
	go debugx.DumpOnSignal(cleanup, syscall.SIGUSR2)

	app := kingpin.New("bearded-wookie", "deployment system").Version(commands.Version)
	agentcmd.configure(app.Command("agent", "agent that manages deployments"))
	client.configure(app.Command("deploy", "deploy to nodes within the cluster"))
	info.configure(app.Command("info", "retrieve info about nodes within the cluster"))
	envinit.configure(app.Command("init", "generate tls cert/key for an environment"))

	if _, err = app.Parse(os.Args[1:]); err != nil {
		log.Fatalf("failed to parse initialization arguments: %+v\n", err)
	}

	systemx.Cleanup(global.ctx, global.shutdown, global.cleanup, os.Kill, os.Interrupt)(func() {
		log.Println("waiting for systems to shutdown")
	})
}
