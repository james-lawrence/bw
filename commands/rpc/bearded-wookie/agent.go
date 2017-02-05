package main

import (
	"log"
	"net"
	"net/rpc"
	"os"

	"bitbucket.org/jatone/bearded-wookie/cluster/serfdom"

	"github.com/alecthomas/kingpin"
	"github.com/pkg/errors"
)

type agent struct {
	*cluster
	network  *net.TCPAddr
	server   *rpc.Server
	listener net.Listener
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

	return t.cluster.Join(
		ctx,
		serfdom.CODelegate(serfdom.NewLocal([]byte{})),
		serfdom.COLogger(os.Stderr),
	)
}
