package main

import (
	"flag"
	"fmt"
	"log"
	"net"
	"net/rpc"
	"strings"

	"github.com/hashicorp/memberlist"

	"bitbucket.org/jatone/bearded-wookie/cluster/serfdom"
	"bitbucket.org/jatone/bearded-wookie/commands/rpc/adapters"
	"bitbucket.org/jatone/bearded-wookie/deployment"
	"bitbucket.org/jatone/bearded-wookie/stringsx"
)

var (
	port       = flag.Int("port", 2000, "port to listen on")
	name       = flag.String("node-name", "", "name of the node, defaults to the network interface")
	ninterface = flag.String("node-network", "127.0.0.1", "network interface to listen on")
	bootstrap  = flag.String("node-bootstrap", "", "bootstrap node")
)

type aliveHandler struct{}

func (aliveHandler) NotifyAlive(peer *memberlist.Node) error {
	log.Println("NotifyAlive", peer.Name)
	if strings.HasPrefix(peer.Name, "lurker") {
		log.Println("NotifyAlive ignoring", peer.Name)
		return fmt.Errorf("ignoring peer: %s", peer.Name)
	}

	return nil
}

type eventHandler struct{}

func (t eventHandler) NotifyJoin(peer *memberlist.Node) {
	log.Println("NotifyJoin", peer.Name)
}

func (t eventHandler) NotifyLeave(peer *memberlist.Node) {
	log.Println("NotifyLeave", peer.Name)
}

func (t eventHandler) NotifyUpdate(peer *memberlist.Node) {
	log.Println("NotifyUpdate", peer.Name)
}

func main() {
	flag.Parse()
	node := stringsx.Default(*name, *ninterface)
	cluster, err := serfdom.New(
		node,
		serfdom.COBindInterface(*ninterface),
		serfdom.COAliveDelegate(aliveHandler{}),
		serfdom.COEventsDelegate(eventHandler{}),
	)

	if err != nil {
		log.Fatal(err)
	}

	defer shutdown(cluster)

	if *bootstrap != "" {
		cluster.Join([]string{*bootstrap})
	}

	addr := fmt.Sprintf("%s:%d", *ninterface, *port)
	fmt.Println("listening on", addr)
	listener, err := net.Listen("tcp", addr)
	if err != nil {
		log.Fatal(err)
	}
	dplyCoordinator, err := deployment.NewDefaultCoordinator()
	if err != nil {
		log.Fatal(err)
	}
	deployments := adapters.Deployment{Coordinator: dplyCoordinator}
	log.Println("Initializing RPC server")

	server := rpc.NewServer()
	server.Register(deployments)
	server.Accept(listener)
}

func shutdown(cluster *memberlist.Memberlist) {
	if err := serfdom.GracefulShutdown(cluster); err != nil {
		log.Println(err)
	}
}
