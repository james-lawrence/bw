package main

import (
	"context"
	"log"
	"net"
	"net/rpc"
	"os"
	"os/signal"
	"sync"
	"syscall"

	"bitbucket.org/jatone/bearded-wookie/commands"
	"bitbucket.org/jatone/bearded-wookie/upnp"
	"bitbucket.org/jatone/bearded-wookie/x/debug"
	"bitbucket.org/jatone/bearded-wookie/x/netx"
	"bitbucket.org/jatone/bearded-wookie/x/systemx"

	"github.com/alecthomas/kingpin"
	"github.com/pkg/errors"
)

type core struct {
	Agent       *agent
	Deployer    *deployer
	upnpEnabled bool
}

type global struct {
	cluster *cluster
	ctx     context.Context
	cancel  context.CancelFunc
	cleanup *sync.WaitGroup
}

// agent: NETWORK=127.0.0.1; ./bin/bearded-wookie agent --agent-bind=$NETWORK:2000 --cluster-bind=$NETWORK:7946
// agent: NETWORK=127.0.0.2; ./bin/bearded-wookie agent --agent-bind=$NETWORK:2000 --cluster-bind=$NETWORK:7946 --cluster-bootstrap=127.0.0.1:7946
// agent: NETWORK=127.0.0.3; ./bin/bearded-wookie agent --agent-bind=$NETWORK:2000 --cluster-bind=$NETWORK:7946 --cluster-bootstrap=127.0.0.1:7946
// agent: NETWORK=127.0.0.4; ./bin/bearded-wookie agent --agent-bind=$NETWORK:2000 --cluster-bind=$NETWORK:7946 --cluster-bootstrap=127.0.0.1:7946
// client: ./bin/bearded-wookie deploy --cluster-node-name="client" --cluster-bootstrap=127.0.0.1:7946 --cluster-bind=127.0.0.1:5000

func main() {
	var (
		err             error
		cleanup, cancel = context.WithCancel(context.Background())
		systemip        = systemx.HostIP(systemx.HostnameOrLocalhost())
		global          = &global{
			cluster: &cluster{
				network: &net.TCPAddr{
					IP:   systemip,
					Port: 7946,
				},
			},
			ctx:     cleanup,
			cancel:  cancel,
			cleanup: &sync.WaitGroup{},
		}
		system = core{
			Agent: &agent{
				global: global,
				network: &net.TCPAddr{
					IP:   systemip,
					Port: 2000,
				},
				listener:    netx.NewNoopListener(),
				server:      rpc.NewServer(),
				upnpEnabled: false,
			},
			Deployer: &deployer{
				global: global,
			},
		}
	)

	app := kingpin.New("bearded-wookie", "deployment system").Version(commands.Version)
	system.Agent.configure(app.Command("agent", "agent that manages deployments").Default())
	system.Deployer.configure(app.Command("deploy", "deploys the application"))

	if _, err = app.Parse(os.Args[1:]); err != nil {
		log.Fatalln("failed to parse initialization arguments:", err)
	}

	go signals(cancel)

	<-cleanup.Done()
	log.Println("waiting for systems to shutdown")
	global.cleanup.Wait()
}

func setupclusterUPNP(c *net.TCPAddr) (*net.TCPAddr, *net.UDPAddr, error) {
	var (
		err            error
		clusterUPNP    *net.TCPAddr
		clusterUPNPUDP *net.UDPAddr
	)
	if clusterUPNPUDP, err = upnp.AddUDP(&net.UDPAddr{IP: c.IP, Port: c.Port}); err != nil {
		return clusterUPNP, clusterUPNPUDP, errors.Wrap(err, "cluster upnp udp failed")
	}

	if clusterUPNP, err = upnp.AddTCP(c); err != nil {
		return clusterUPNP, clusterUPNPUDP, errors.Wrap(err, "cluster upnp udp failed")
	}

	return clusterUPNP, clusterUPNPUDP, nil
}

func setUDPRecvBuf(c *net.UDPConn, size int) {
	for {
		if err := c.SetReadBuffer(size); err == nil {
			break
		}
		size = size / 2
	}
}

func signals(shutdown context.CancelFunc) {
	signals := make(chan os.Signal, 1)
	signal.Notify(signals, os.Kill, os.Interrupt, syscall.SIGUSR2)
	defer close(signals)
	defer signal.Stop(signals)

	for s := range signals {
		switch s {
		case os.Kill, os.Interrupt:
			log.Println("shutdown request received")
			shutdown()
		case syscall.SIGUSR2:
			var (
				err  error
				path string
			)

			if path, err = debug.DumpRoutines(); err != nil {
				log.Println("failed to dump routines:", err)
			} else {
				log.Println("dump located at:", path)
			}
		}
	}
}
