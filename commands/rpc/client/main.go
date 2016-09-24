package main

import (
	"fmt"
	"log"
	"net"
	"net/rpc"
	"os"
	"strings"
	"time"

	"github.com/hashicorp/memberlist"
	"gopkg.in/alecthomas/kingpin.v2"

	"bitbucket.org/jatone/bearded-wookie/cluster/serfdom"
	"bitbucket.org/jatone/bearded-wookie/commands/rpc/adapters"
	"bitbucket.org/jatone/bearded-wookie/deployment"
)

func main() {
	var (
		err            error
		actual         string
		clusterAddress = &net.TCPAddr{}
		local          = &net.TCPAddr{}
	)

	app := kingpin.New("client", "client for interacting with a deployment cluster")
	app.Flag("cluster-address", "cluster server address").Default("127.0.0.1:7946").TCPVar(&clusterAddress)
	app.Flag("cluster-local-address", "local cluster network to bind").Default("localhost:5001").TCPVar(&local)

	deploy := app.Command("deploy", "deploys the application")
	all := deploy.Command("all", "deploy to all nodes within the cluster").Default()

	if actual, err = app.Parse(os.Args[1:]); err != nil {
		log.Fatalln("failed to parse initialization arguments:", err)
	}

	fmt.Println("creating serf client")
	cluster, err := serfdom.New("lurker-client",
		serfdom.COBindInterface(local.IP.String()),
		serfdom.COBindPort(local.Port),
		serfdom.COAliveDelegate(aliveHandler{}),
		serfdom.COEventsDelegate(eventHandler{}),
	)
	if err != nil {
		log.Fatal(err)
	}
	defer func() {
		var (
			err error
		)

		if err = cluster.Leave(5 * time.Second); err != nil {
			log.Println("failure to leave cluster", err)
		}

		if err = cluster.Shutdown(); err != nil {
			log.Println("failure to shutdown node", err)
		}
	}()

	fmt.Println("joining serf cluster")

	_, err = cluster.Join([]string{clusterAddress.String()})
	if err != nil {
		log.Panic(err)
	}

	switch actual {
	case all.FullCommand():
		deployment.Deploy(cluster, deployer, status)
	}

	fmt.Println("joined a cluster of size", cluster.NumMembers())
}

func status(peer *memberlist.Node) error {
	rpcClient, err := rpc.Dial("tcp", net.JoinHostPort(peer.Addr.String(), "2000"))
	if err != nil {
		log.Println("failed to connect to", peer.Name, err)
		return err
	}
	defer rpcClient.Close()
	deployClient := adapters.DeploymentClient{Client: rpcClient}
	return deployClient.Status()
}

func deployer(peer *memberlist.Node) error {
	rpcClient, err := rpc.Dial("tcp", net.JoinHostPort(peer.Addr.String(), "2000"))
	if err != nil {
		return err
	}

	return adapters.DeploymentClient{Client: rpcClient}.Deploy()
}

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
