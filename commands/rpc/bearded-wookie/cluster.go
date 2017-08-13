package main

import (
	"fmt"
	"log"
	"net"

	cp "bitbucket.org/jatone/bearded-wookie/cluster"
	"bitbucket.org/jatone/bearded-wookie/clustering"
	"bitbucket.org/jatone/bearded-wookie/clustering/peering"
	"bitbucket.org/jatone/bearded-wookie/x/stringsx"

	"github.com/alecthomas/kingpin"
	"github.com/hashicorp/memberlist"
	"github.com/pkg/errors"
)

type cluster struct {
	name                string
	network             *net.TCPAddr
	bootstrap           []*net.TCPAddr
	singleNodeOperation bool
}

func (t *cluster) configure(parent *kingpin.CmdClause) {
	parent.Flag("cluster-node-name", "name of the node within the cluster").StringVar(&t.name)
	parent.Flag("cluster-bootstrap", "addresses to bootstrap the cluster from").TCPListVar(&t.bootstrap)
	parent.Flag("cluster-bind", "address to bind").Default(t.network.String()).TCPVar(&t.network)
	parent.Flag("cluster-single-node", "enable single mode operation").BoolVar(&t.singleNodeOperation)
}

func (t *cluster) Join(options ...clustering.Option) (clustering.Cluster, error) {
	var (
		err error
		c   clustering.Cluster
	)

	log.Println("connecting to cluster")
	defer log.Println("connection to cluster complete")

	defaults := []clustering.Option{
		clustering.OptionNodeID(stringsx.DefaultIfBlank(t.name, t.network.IP.String())),
		clustering.OptionBindAddress(t.network.IP.String()),
		clustering.OptionBindPort(t.network.Port),
		clustering.OptionEventDelegate(eventHandler{}),
		clustering.OptionAliveDelegate(aliveHandler{}),
	}

	options = append(defaults, options...)
	if c, err = clustering.NewOptions(options...).NewCluster(); err != nil {
		return c, errors.Wrap(err, "failed to join cluster")
	}

	peerings := clustering.BootstrapOptionPeeringStrategies(
		peering.Closure(func() ([]string, error) {
			addresses := make([]string, 0, len(t.bootstrap))
			for _, addr := range t.bootstrap {
				addresses = append(addresses, addr.String())
			}

			return addresses, nil
		}),
	)

	if err = clustering.Bootstrap(c, peerings); err != nil {
		if !t.singleNodeOperation {
			return c, errors.Wrap(err, "failed to bootstrap cluster")
		}
	}

	return c, nil
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
