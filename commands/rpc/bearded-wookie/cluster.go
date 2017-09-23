package main

import (
	"context"
	"crypto/tls"
	"log"
	"net"
	"os"
	"path/filepath"
	"time"

	"bitbucket.org/jatone/bearded-wookie/agent"
	xcluster "bitbucket.org/jatone/bearded-wookie/cluster"
	"bitbucket.org/jatone/bearded-wookie/clustering"
	"bitbucket.org/jatone/bearded-wookie/clustering/peering"
	"bitbucket.org/jatone/bearded-wookie/clustering/raftutil"
	"bitbucket.org/jatone/bearded-wookie/x/stringsx"

	"github.com/alecthomas/kingpin"
	"github.com/hashicorp/memberlist"
	"github.com/hashicorp/raft"
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

func clusterCmdOptionRaftBind(b *net.TCPAddr) clusterCmdOption {
	return func(c *cluster) {
		c.raftNetwork = b
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
	raftNetwork          *net.TCPAddr
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
	parent.Flag("cluster-bind-raft", "address for the raft protocol to bind to").Default(t.raftNetwork.String()).TCPVar(&t.raftNetwork)
	parent.Flag("cluster-minimum-required-peers", "minimum number of peers required to join the cluster").Default("1").IntVar(&t.minimumRequiredPeers)
	parent.Flag("cluster-maximum-join-attempts", "maximum number of times to attempt to join the cluster").Default("1").IntVar(&t.maximumAttempts)
}

func (t *cluster) Join(snap peering.File, options ...clustering.Option) (clustering.Cluster, error) {
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
		snap,
	)

	if err = clustering.Bootstrap(c, peerings, joins, attempts); err != nil {
		return c, errors.Wrap(err, "failed to bootstrap cluster")
	}

	return c, nil
}

func (t *cluster) Snapshot(c clustering.Cluster, fssnapshot peering.File, options ...clustering.SnapshotOption) {
	go clustering.Snapshot(
		c,
		fssnapshot,
		options...,
	)
}

func (t *cluster) Raft(ctx context.Context, conf agent.Config) (p raftutil.Protocol, err error) {
	var (
		cs      *tls.Config
		l       net.Listener
		snaps   raft.SnapshotStore
		streaml *raft.NetworkTransport
	)

	if cs, err = conf.TLSConfig.BuildServer(); err != nil {
		return p, errors.WithStack(err)
	}

	if snaps, err = raft.NewFileSnapshotStore(filepath.Join(conf.Root, "raft"), 2, nil); err != nil {
		return p, errors.WithStack(err)
	}

	if l, err = net.ListenTCP(t.raftNetwork.Network(), t.raftNetwork); err != nil {
		return p, errors.WithStack(err)
	}
	streaml = raft.NewNetworkTransport(raftutil.NewTLSStreamLayer(t.raftNetwork.Port, l, cs), 3, 2*time.Second, os.Stderr)

	return raftutil.NewProtocol(
		ctx,
		uint16(t.raftNetwork.Port),
		streaml,
		snaps,
	)
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
