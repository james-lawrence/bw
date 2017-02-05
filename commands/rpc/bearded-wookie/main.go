package main

import (
	"context"
	"log"
	"net"
	"net/rpc"
	"os"
	"os/signal"
	"syscall"
	"time"

	"bitbucket.org/jatone/bearded-wookie/cluster/serfdom"
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

// agent: NETWORK=127.0.0.1; ./bin/bearded-wookie agent --agent-bind=$NETWORK:2000 --cluster-bind=$NETWORK:7946
// agent: NETWORK=127.0.0.2; ./bin/bearded-wookie agent --agent-bind=$NETWORK:2000 --cluster-bind=$NETWORK:7946 --cluster-bootstrap=127.0.0.1:7946
// agent: NETWORK=127.0.0.3; ./bin/bearded-wookie agent --agent-bind=$NETWORK:2000 --cluster-bind=$NETWORK:7946 --cluster-bootstrap=127.0.0.1:7946
// agent: NETWORK=127.0.0.4; ./bin/bearded-wookie agent --agent-bind=$NETWORK:2000 --cluster-bind=$NETWORK:7946 --cluster-bootstrap=127.0.0.1:7946
// client: ./bin/bearded-wookie deploy --cluster-node-name="client" --cluster-bootstrap=127.0.0.1:7946 --cluster-bind=127.0.0.1:5000

func main() {
	var (
		err             error
		agentUPNP       *net.TCPAddr
		clusterUPNP     *net.TCPAddr
		clusterUPNPUDP  *net.UDPAddr
		cleanup, cancel = context.WithCancel(context.Background())
		systemip        = systemx.HostIP(systemx.HostnameOrLocalhost())
		c               = &cluster{
			network: &net.TCPAddr{
				IP:   systemip,
				Port: 7946,
			},
		}
		system = core{
			Agent: &agent{
				network: &net.TCPAddr{
					IP:   systemip,
					Port: 2000,
				},
				cluster:     c,
				listener:    netx.NewNoopListener(),
				server:      rpc.NewServer(),
				upnpEnabled: false,
			},
			Deployer: &deployer{
				cluster: c,
				ctx:     cleanup,
				cancel:  cancel,
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
	clusterOptions := []serfdom.ClusterOption{
		serfdom.CODelegate(serfdom.NewLocal([]byte{})),
		serfdom.COLogger(os.Stderr),
	}

	if system.Agent.upnpEnabled {
		if agentUPNP, err = upnp.AddTCP(system.Agent.network); err != nil {
			log.Println("agent upnp failed", err)
		}

		if clusterUPNP, clusterUPNPUDP, err = setupclusterUPNP(system.Agent.cluster.network); err != nil {
			log.Println("failed to setup cluster upnp", err)
			clusterOptions = append(
				clusterOptions,
				serfdom.COAdvertiseInterface("73.119.50.204"),
				serfdom.COAdvertisePort(system.Agent.cluster.network.Port),
			)
		} else {
			clusterOptions = append(
				clusterOptions,
				serfdom.COAdvertiseInterface(clusterUPNP.IP.String()),
				serfdom.COAdvertisePort(clusterUPNP.Port),
			)
		}
	}
	err = system.Agent.cluster.Join(nil, clusterOptions...)

	if err != nil {
		log.Println("failed to join cluster", err)
		cancel()
	}

	go func() {
		for _ = range time.Tick(time.Second) {
			log.Println("cluster size", system.Agent.cluster.memberlist.NumMembers())
		}
	}()
	<-cleanup.Done()

	if system.Agent.upnpEnabled {
		log.Println("agent upnp", upnp.DeleteTCP(agentUPNP))
		log.Println("cluster upnp", upnp.DeleteTCP(clusterUPNP))
		log.Println("cluster upnp udp", upnp.DeleteUDP(clusterUPNPUDP))
	}

	log.Println("left cluster", system.Agent.cluster.memberlist.Leave(5*time.Second))
	log.Println("cluster shutdown", system.Agent.cluster.memberlist.Shutdown())
	log.Println("agent shutdown", system.Agent.listener.Close())
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
