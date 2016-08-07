package main

import (
	"flag"
	"fmt"
	"log"
	"net"
	"net/rpc"

	"bitbucket.org/jatone/bearded-wookie/cluster/serfdom"
	"bitbucket.org/jatone/bearded-wookie/commands/rpc/adapters"
	"bitbucket.org/jatone/bearded-wookie/deployment"
)

var port = flag.Int("port", 2000, "port to listen on")

func main() {
	flag.Parse()
	_, err := serfdom.NewDefault("node1", "0.0.0.0", 5000)
	if err != nil {
		log.Fatal(err)
	}

	addr := fmt.Sprintf("0.0.0.0:%d", *port)
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
