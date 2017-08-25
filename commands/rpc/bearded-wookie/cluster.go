package main

import (
	"log"
	"net"

	xcluster "bitbucket.org/jatone/bearded-wookie/cluster"
	"bitbucket.org/jatone/bearded-wookie/clustering"
	"bitbucket.org/jatone/bearded-wookie/clustering/peering"
	"bitbucket.org/jatone/bearded-wookie/x/stringsx"

	"github.com/alecthomas/kingpin"
	"github.com/hashicorp/memberlist"
	"github.com/pkg/errors"
)

type clusterCmdOption func(*cluster)

func clusterCmdOptionAddress(addresses ...*net.TCPAddr) clusterCmdOption {
	return func(c *cluster) {
		c.bootstrap = addresses
	}
}

func clusterCmdOptionBind(b *net.TCPAddr) clusterCmdOption {
	return func(c *cluster) {
		c.network = b
	}
}

func clusterCmdOptionMinPeers(b int) clusterCmdOption {
	return func(c *cluster) {
		c.minimumRequiredPeers = 1
	}
}

type cluster struct {
	name                 string
	network              *net.TCPAddr
	bootstrap            []*net.TCPAddr
	minimumRequiredPeers int
	maximumAttempts      int
}

func (t *cluster) fromOptions(options ...clusterCmdOption) {
	for _, opt := range options {
		opt(t)
	}
}

func (t *cluster) configure(parent *kingpin.CmdClause, options ...clusterCmdOption) {
	t.fromOptions(options...)
	parent.Flag("cluster-node-name", "name of the node within the cluster").Default(t.network.String()).StringVar(&t.name)
	parent.Flag("cluster", "addresses of the cluster to connect to").TCPListVar(&t.bootstrap)
	parent.Flag("cluster-bind", "address to bind").Default(t.network.String()).TCPVar(&t.network)
	parent.Flag("cluster-minimum-required-peers", "minimum number of peers required to join the cluster").Default("1").IntVar(&t.minimumRequiredPeers)
	parent.Flag("cluster-maximum-join-attempts", "maximum number of times to attempt to join the cluster").Default("1").IntVar(&t.maximumAttempts)
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
		clustering.OptionAliveDelegate(xcluster.AliveDefault{}),
	}

	options = append(defaults, options...)
	if c, err = clustering.NewOptions(options...).NewCluster(); err != nil {
		return c, errors.Wrap(err, "failed to join cluster")
	}

	joins := clustering.BootstrapOptionJoinStrategy(clustering.MinimumPeers(t.minimumRequiredPeers))
	attempts := clustering.BootstrapOptionAllowRetry(clustering.MaximumAttempts(t.maximumAttempts))
	peerings := clustering.BootstrapOptionPeeringStrategies(
		peering.Closure(func() ([]string, error) {
			addresses := make([]string, 0, len(t.bootstrap))
			for _, addr := range t.bootstrap {
				addresses = append(addresses, addr.String())
			}

			return addresses, nil
		}),
	)

	if err = clustering.Bootstrap(c, peerings, joins, attempts); err != nil {
		return c, errors.Wrap(err, "failed to bootstrap cluster")
	}

	return c, nil
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
