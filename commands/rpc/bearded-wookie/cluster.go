package main

import (
	"fmt"
	"log"
	"net"

	cp "bitbucket.org/jatone/bearded-wookie/cluster"
	"bitbucket.org/jatone/bearded-wookie/cluster/serfdom"
	"bitbucket.org/jatone/bearded-wookie/x/stringsx"

	"github.com/alecthomas/kingpin"
	"github.com/hashicorp/memberlist"
	"github.com/pkg/errors"
)

type cluster struct {
	name       string
	network    *net.TCPAddr
	bootstrap  []*net.TCPAddr
	memberlist *memberlist.Memberlist
}

func (t *cluster) configure(parent *kingpin.CmdClause) {
	parent.Flag("cluster-node-name", "name of the node within the cluster").StringVar(&t.name)
	parent.Flag("cluster-bootstrap", "addresses to bootstrap the cluster from").TCPListVar(&t.bootstrap)
	parent.Flag("cluster-bind", "address to bind").Default(t.network.String()).TCPVar(&t.network)
}

func (t *cluster) Join(_ *kingpin.ParseContext, options ...serfdom.ClusterOption) error {
	var (
		err    error
		joined int
	)

	log.Println("connecting to cluster")
	defer log.Println("connection to cluster complete")

	defaults := []serfdom.ClusterOption{
		serfdom.COBindInterface(t.network.IP.String()),
		serfdom.COBindPort(t.network.Port),
		serfdom.COEventsDelegate(eventHandler{}),
		serfdom.COAliveDelegate(aliveHandler{}),
	}

	t.memberlist, err = serfdom.New(
		stringsx.DefaultIfBlank(t.name, t.network.IP.String()),
		append(defaults, options...)...,
	)

	if err != nil {
		return err
	}

	addresses := make([]string, 0, len(t.bootstrap))
	for _, addr := range t.bootstrap {
		addresses = append(addresses, addr.String())
	}

	if joined, err = t.memberlist.Join(addresses); err != nil {
		return errors.Wrapf(err, "failed to join cluster %s", addresses)
	}

	log.Println("joined a cluster with", joined, "node(s)")
	return nil
}

type aliveHandler struct{}

func (aliveHandler) NotifyAlive(peer *memberlist.Node) error {
	log.Printf("NotifyAlive peer %s metadata %s\n", peer.Name, peer.Meta)
	if cp.BitField(peer.Meta).Has(cp.Lurker) {
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
