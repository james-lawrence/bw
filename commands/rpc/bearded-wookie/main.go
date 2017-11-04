package main

import (
	"context"
	"log"
	"net"
	"os"
	"path/filepath"
	"sync"
	"syscall"

	"bitbucket.org/jatone/bearded-wookie"
	"bitbucket.org/jatone/bearded-wookie/agent"
	"bitbucket.org/jatone/bearded-wookie/commands"
	"bitbucket.org/jatone/bearded-wookie/x/debugx"
	"bitbucket.org/jatone/bearded-wookie/x/netx"
	"bitbucket.org/jatone/bearded-wookie/x/systemx"

	"github.com/alecthomas/kingpin"
	"github.com/gizak/termui"
)

type global struct {
	systemIP net.IP
	cluster  *clusterCmd
	ctx      context.Context
	shutdown context.CancelFunc
	cleanup  *sync.WaitGroup
}

// agent: NETWORK=127.0.0.1; ./bin/bearded-wookie agent --agent-name="node1" --agent-bind=$NETWORK:2000 --cluster-bind=$NETWORK:2001 --cluster-bind-raft=$NETWORK:2002 --cluster=127.0.0.2:2001 --cluster-minimum-required-peers=1 --cluster-maximum-join-attempts=10 --agent-config=".bwagent1/agent.config"
// agent: NETWORK=127.0.0.2; ./bin/bearded-wookie agent --agent-name="node2" --agent-bind=$NETWORK:2000 --cluster-bind=$NETWORK:2001 --cluster-bind-raft=$NETWORK:2002 --cluster=127.0.0.1:2001 --cluster-minimum-required-peers=1 --cluster-maximum-join-attempts=10 --agent-config=".bwagent2/agent.config"
// agent: NETWORK=127.0.0.3; ./bin/bearded-wookie agent --agent-name="node3" --agent-bind=$NETWORK:2000 --cluster-bind=$NETWORK:2001 --cluster-bind-raft=$NETWORK:2002 --cluster=127.0.0.1:2001 --cluster-minimum-required-peers=1 --cluster-maximum-join-attempts=10 --agent-config=".bwagent3/agent.config"
// agent: NETWORK=127.0.0.4; ./bin/bearded-wookie agent --agent-name="node4" --agent-bind=$NETWORK:2000 --cluster-bind=$NETWORK:2001 --cluster-bind-raft=$NETWORK:2002 --cluster=127.0.0.1:2001 --cluster-minimum-required-peers=1 --cluster-maximum-join-attempts=10 --agent-config=".bwagent4/agent.config"
// client: ./bin/bearded-wookie deploy

// [agents] -> peers within the cluster
// [quorum] -> subset of agents responsible for managing cluster state
// [client] -> perform actions within the cluster.

// order of precedence for options: environment overrides command line overrides configuration file.
func main() {
	var (
		err             error
		cleanup, cancel = context.WithCancel(context.Background())
		systemip        = systemx.HostIP(systemx.HostnameOrLocalhost())
		global          = &global{
			systemIP: systemx.HostIP(systemx.HostnameOrLocalhost()),
			cluster:  &clusterCmd{},
			ctx:      cleanup,
			shutdown: cancel,
			cleanup:  &sync.WaitGroup{},
		}

		agentcmd = &agentCmd{
			config:   agent.NewConfig(agent.ConfigOptionDefaultBind(systemip)),
			global:   global,
			listener: netx.NewNoopListener(),
		}
		client = &deployCmd{
			global: global,
		}
		info = &agentInfo{
			node:   bw.MustGenerateID().String(),
			global: global,
		}
		envinit = &initCmd{
			global: global,
		}
	)

	log.SetFlags(log.Flags() | log.Lshortfile)
	go debugx.DumpOnSignal(cleanup, syscall.SIGUSR2)
	go systemx.Cleanup(global.ctx, global.shutdown, global.cleanup, os.Kill, os.Interrupt)(func() {
		termui.Close()
		termui.Clear()
		log.Println("waiting for systems to shutdown")
	})
	app := kingpin.New("bearded-wookie", "deployment system").Version(commands.Version)
	agentcmd.configure(app.Command("agent", "agent that manages deployments"))
	client.configure(app.Command("deploy", "deploy to nodes within the cluster"))
	info.configure(app.Command("info", "retrieve info about nodes within the cluster"))
	envinit.configure(app.Command("init", "generate tls cert/key for an environment"))

	if _, err = app.Parse(os.Args[1:]); err != nil {
		log.Fatalf("failed to parse initialization arguments: %+v\n", err)
	}

	global.cleanup.Wait()
}

func loadConfiguration(environment string) (agent.ConfigClient, error) {
	path := filepath.Join(bw.LocateDeployspace(bw.DefaultDeployspaceConfigDir), environment)
	log.Println("loading configuration", path)
	return agent.DefaultConfigClient(agent.CCOptionTLSConfig(environment)).LoadConfig(path)
}
