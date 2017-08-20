package main

import (
	"context"
	"log"
	"net"
	"os"
	"os/user"
	"sync"
	"syscall"

	"bitbucket.org/jatone/bearded-wookie/commands"
	"bitbucket.org/jatone/bearded-wookie/x/debugx"
	"bitbucket.org/jatone/bearded-wookie/x/netx"
	"bitbucket.org/jatone/bearded-wookie/x/systemx"

	"github.com/alecthomas/kingpin"
)

const (
	workspaceDefault      = ".bw"
	configDirDefault      = ".bwconfig"
	credentialsDirDefault = ".bwcreds"
	credentialsDefault    = "default"
	environmentDefault    = "default"
	tlscaKeyDefault       = "tlsca.key"
	tlscaCertDefault      = "tlsca.cert"
	tlsclientKeyDefault   = "tlsclient.key"
	tlsclientCertDefault  = "tlsclient.cert"
	tlsserverKeyDefault   = "tlsserver.key"
	tlsserverCertDefault  = "tlsserver.cert"
)

type global struct {
	systemIP net.IP
	cluster  *cluster
	ctx      context.Context
	shutdown context.CancelFunc
	cleanup  *sync.WaitGroup
	user     *user.User
}

// agent: NETWORK=127.0.0.1; ./bin/bearded-wookie agent --agent-bind=$NETWORK:2000 --cluster-bind=$NETWORK:7946 --cluster-maximum-join-attempts=10
// agent: NETWORK=127.0.0.2; ./bin/bearded-wookie agent --agent-bind=$NETWORK:2000 --cluster-bind=$NETWORK:7946 --cluster-bootstrap=127.0.0.1:7946 --cluster-maximum-join-attempts=10
// agent: NETWORK=127.0.0.3; ./bin/bearded-wookie agent --agent-bind=$NETWORK:2000 --cluster-bind=$NETWORK:7946 --cluster-bootstrap=127.0.0.1:7946 --cluster-maximum-join-attempts=10
// agent: NETWORK=127.0.0.4; ./bin/bearded-wookie agent --agent-bind=$NETWORK:2000 --cluster-bind=$NETWORK:7946 --cluster-bootstrap=127.0.0.1:7946 --cluster-maximum-join-attempts=10
// client: ./bin/bearded-wookie deploy --cluster-node-name="client" --cluster-bootstrap=127.0.0.1:7946 --cluster-bind=127.0.0.1:5000

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
			user:     systemx.MustUser(),
		}
		agent = &agentCmd{
			global: global,
			network: &net.TCPAddr{
				IP:   systemip,
				Port: 2000,
			},
			listener:    netx.NewNoopListener(),
			upnpEnabled: false,
		}
		client = &deployCmd{
			global: global,
		}
		envinit = &initCmd{
			global: global,
		}
	)

	log.SetFlags(log.Flags() | log.Lshortfile)
	go debugx.DumpOnSignal(cleanup, syscall.SIGUSR2)

	app := kingpin.New("bearded-wookie", "deployment system").Version(commands.Version)
	agent.configure(app.Command("agent", "agent that manages deployments"))
	client.deployCmd(app.Command("deploy", "deploy to all nodes within the cluster").Default())
	client.filteredCmd(app.Command("filtered", "allows for filtering the instances within the cluster"))
	envinit.configure(app.Command("init", "generate tls cert/key for an environment"))

	if _, err = app.Parse(os.Args[1:]); err != nil {
		log.Fatalf("failed to parse initialization arguments: %+v\n", err)
	}

	systemx.Cleanup(global.ctx, global.shutdown, global.cleanup, os.Kill, os.Interrupt)(func() {
		log.Println("waiting for systems to shutdown")
	})
}
