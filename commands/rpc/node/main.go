package main

import (
	"fmt"
	"log"
	"net"
	"net/rpc"
	"os"
	"strconv"
	"strings"

	"github.com/hashicorp/memberlist"

	"bitbucket.org/jatone/bearded-wookie/cluster/serfdom"
	"bitbucket.org/jatone/bearded-wookie/stringsx"

	"gopkg.in/alecthomas/kingpin.v2"
)

type rpcconfig struct {
	port     int
	server   *rpc.Server
	listener net.Listener
}

type core struct {
	name    string
	network *net.TCPAddr
	rpc     rpcconfig
	cluster struct {
		port       int
		memberlist *memberlist.Memberlist
	}
}

func main() {
	var (
		err    error
		system = core{
			name:    "",
			network: &net.TCPAddr{},
			rpc: rpcconfig{
				server: rpc.NewServer(),
			},
		}
		shutdown chan struct{}
	)

	app := kingpin.New("node", "node in the deployment cluster")
	app.Flag("node-network", "network interface to listen on").Default("127.0.0.1:").TCPVar(&system.network)
	app.Flag("node-port", "port for the cluster to listen on").Default("7946").IntVar(&system.cluster.port)
	app.Flag("node-name", "name to give to the node, defaults to the node-network").StringVar(&system.name)
	app.Flag("rpc-port", "port for the rpc server to listen on").Default("2000").IntVar(&system.rpc.port)
	app.Action(rpcBind(&system))

	if err = configure(&system, app); err != nil {
		log.Fatalln("failed to configure application", err)
	}

	if _, err = app.Parse(os.Args[1:]); err != nil {
		log.Fatalln("failed to parse initialization arguments:", err)
	}
	log.Println("parsing completed")

	go system.rpc.server.Accept(system.rpc.listener)
	defer system.rpc.listener.Close()
	log.Println("rpc server awaiting requests")

	clusterConnect(&system)
	defer clusterShutdown(system.cluster.memberlist)

	log.Println("awaiting shutdown")
	<-shutdown
}

func rpcBind(c *core) kingpin.Action {
	return func(*kingpin.ParseContext) error {
		var (
			err error
		)
		addr := net.JoinHostPort(
			c.network.IP.String(),
			strconv.Itoa(c.rpc.port),
		)

		log.Println("binding rpc server", addr)
		defer log.Println("done binding rpc server")
		c.rpc.listener, err = net.Listen(
			"tcp",
			addr,
		)

		return err
	}
}

func clusterConnect(c *core) {
	var (
		err error
	)

	log.Println("connecting to cluster")
	defer log.Println("connection to cluster complete")

	c.cluster.memberlist, err = serfdom.New(
		stringsx.Default(c.name, c.network.IP.String()),
		serfdom.COBindInterface(c.network.IP.String()),
		serfdom.COBindPort(c.cluster.port),
		serfdom.COAliveDelegate(aliveHandler{}),
		serfdom.COEventsDelegate(eventHandler{}),
	)

	if err != nil {
		log.Fatalln(err)
	}
}

func clusterShutdown(cluster *memberlist.Memberlist) {
	if err := serfdom.GracefulShutdown(cluster); err != nil {
		log.Println(err)
	}
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
