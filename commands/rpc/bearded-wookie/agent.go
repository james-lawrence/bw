package main

import (
	"log"
	"net"
	"os"
	"strconv"

	"google.golang.org/grpc"

	"bitbucket.org/jatone/bearded-wookie/cluster/serfdom"
	"bitbucket.org/jatone/bearded-wookie/clustering"

	"github.com/alecthomas/kingpin"
	"github.com/pkg/errors"
)

type agentCmd struct {
	*global
	network     *net.TCPAddr
	server      *grpc.Server
	listener    net.Listener
	upnpEnabled bool
}

func (t *agentCmd) configure(parent *kingpin.CmdClause) {
	t.cluster.configure(
		parent,
		clusterCmdOptionBind(
			&net.TCPAddr{
				IP:   t.global.systemIP,
				Port: 2001,
			},
		),
	)
	parent.Flag("upnp-enabled", "enable upnp forwarding for the agent").Default(strconv.FormatBool(t.upnpEnabled)).Hidden().BoolVar(&t.upnpEnabled)
	parent.Flag("agent-bind", "network interface to listen on").Default(t.network.String()).TCPVar(&t.network)

	t.operatingSystemSpecificConfiguration(parent)
}

func (t *agentCmd) Bind(ctx *kingpin.ParseContext) error {
	var (
		err error
		c   clustering.Cluster
	)

	log.Println("initiated binding rpc server", t.network.String())
	defer log.Println("completed binding rpc server", t.network.String())

	if t.listener, err = net.Listen("tcp", t.network.String()); err != nil {
		return errors.Wrapf(err, "failed to bind agent to %s", t.network)
	}

	options := []clustering.Option{
		clustering.OptionDelegate(serfdom.NewLocal([]byte{})),
		clustering.OptionLogger(os.Stderr),
	}

	if c, err = t.global.cluster.Join(options...); err != nil {
		return errors.Wrap(err, "failed to join cluster")
	}

	go t.server.Serve(t.listener)

	t.global.cleanup.Add(1)
	go func() {
		defer t.global.cleanup.Done()
		<-t.global.ctx.Done()

		log.Println("left cluster", c.Shutdown())
		log.Println("agent shutdown", t.listener.Close())
	}()

	return nil
}
