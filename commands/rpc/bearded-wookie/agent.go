package main

import (
	"log"
	"net"
	"net/rpc"
	"strconv"

	"github.com/alecthomas/kingpin"
	"github.com/pkg/errors"
)

type agent struct {
	*cluster
	network     *net.TCPAddr
	server      *rpc.Server
	listener    net.Listener
	upnpEnabled bool
}

func (t *agent) configure(parent *kingpin.CmdClause) {
	t.cluster.configure(parent)
	parent.Flag("upnp-enabled", "enable upnp forwarding for the agent").Default(strconv.FormatBool(t.upnpEnabled)).Hidden().BoolVar(&t.upnpEnabled)
	parent.Flag("agent-bind", "network interface to listen on").Default(t.network.String()).TCPVar(&t.network)
	parent.Action(t.Bind)
	t.operatingSystemSpecificConfiguration(parent)
}

func (t *agent) Bind(ctx *kingpin.ParseContext) error {
	var (
		err error
	)

	log.Println("initiated binding rpc server", t.network.String())
	defer log.Println("completed binding rpc server", t.network.String())

	if t.listener, err = net.Listen("tcp", t.network.String()); err != nil {
		return errors.Wrapf(err, "failed to bind agent to %s", t.network)
	}

	go t.server.Accept(t.listener)

	return nil
}
