package main

import (
	"log"
	"net"
	"net/rpc"

	"bitbucket.org/jatone/bearded-wookie/cluster/serfdom"

	"github.com/pkg/errors"
	"gopkg.in/alecthomas/kingpin.v2"
)

type agent struct {
	*cluster
	network  *net.TCPAddr
	server   *rpc.Server
	listener net.Listener
}

func (t *agent) configure(parent *kingpin.CmdClause) {
	t.cluster.configure(parent)
	parent.Flag("agent-bind", "network interface to listen on").Default("127.0.0.1:2000").TCPVar(&t.network)
	parent.Action(t.Bind)

	(&dummy{agent: t}).configure(parent.Command("dummy", "dummy deployment").Default())
	(&packagekit{agent: t}).configure(parent.Command("packagekit", "packagekit deployment"))
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

	return t.cluster.Join(ctx, serfdom.CODelegate(serfdom.NewLocal([]byte{})))
}
