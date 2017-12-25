package main

import (
	"context"
	"crypto/tls"
	"log"
	"math"
	"net"
	"strconv"

	"github.com/james-lawrence/bw/agent"
	"github.com/james-lawrence/bw/awsx"
	"github.com/james-lawrence/bw/cluster"
	"github.com/james-lawrence/bw/clustering"
	"github.com/james-lawrence/bw/clustering/peering"
	"github.com/james-lawrence/bw/clustering/raftutil"

	"github.com/alecthomas/kingpin"
	"github.com/hashicorp/raft"
	"github.com/pkg/errors"
)

type clusterCmdOption func(*clusterCmd)

func clusterCmdOptionName(n string) clusterCmdOption {
	return func(c *clusterCmd) {
		c.name = n
	}
}

func clusterCmdOptionAddress(addresses ...*net.TCPAddr) clusterCmdOption {
	return func(c *clusterCmd) {
		c.bootstrap = addresses
	}
}

func clusterCmdOptionBind(b *net.TCPAddr) clusterCmdOption {
	return func(c *clusterCmd) {
		c.swimNetwork = b
	}
}

func clusterCmdOptionRaftBind(b *net.TCPAddr) clusterCmdOption {
	return func(c *clusterCmd) {
		c.raftNetwork = b
	}
}

func clusterCmdOptionMinPeers(b int) clusterCmdOption {
	return func(c *clusterCmd) {
		c.minimumRequiredNodes = b
	}
}

type clusterCmd struct {
	name                 string
	swimNetwork          *net.TCPAddr
	raftNetwork          *net.TCPAddr
	bootstrap            []*net.TCPAddr
	minimumRequiredNodes int
	maximumAttempts      int
	awsBootstrap         bool
}

func (t *clusterCmd) fromOptions(options ...clusterCmdOption) {
	for _, opt := range options {
		opt(t)
	}
}

func (t *clusterCmd) configure(parent *kingpin.CmdClause, options ...clusterCmdOption) {
	t.fromOptions(options...)
	parent.Flag("cluster", "addresses of the cluster to bootstrap from").PlaceHolder(t.swimNetwork.String()).TCPListVar(&t.bootstrap)
	parent.Flag("cluster-bind", "address for the swim protocol (cluster membership) to bind to").PlaceHolder(t.swimNetwork.String()).TCPVar(&t.swimNetwork)
	parent.Flag("cluster-bind-raft", "address for the raft protocol to bind to").PlaceHolder(t.raftNetwork.String()).TCPVar(&t.raftNetwork)
	parent.Flag("cluster-minimum-required-peers", "minimum number of peers required to join the cluster").Default("1").IntVar(&t.minimumRequiredNodes)
	parent.Flag("cluster-maximum-join-attempts", "maximum number of times to attempt to join the cluster").Default(strconv.Itoa(math.MaxInt32)).IntVar(&t.maximumAttempts)
	parent.Flag("cluster-asg", "enable the aws autoscale group bootstrapping strategy").Default("false").BoolVar(&t.awsBootstrap)
}

func (t *clusterCmd) Join(snap peering.File, options ...clustering.Option) (clustering.Cluster, error) {
	type peers interface {
		Peers() ([]string, error)
	}
	var (
		err error
		c   clustering.Cluster
	)

	log.Println("connecting to cluster")
	defer log.Println("connection to cluster complete")

	defaults := []clustering.Option{
		clustering.OptionBindAddress(t.swimNetwork.IP.String()),
		clustering.OptionBindPort(t.swimNetwork.Port),
		clustering.OptionEventDelegate(cluster.LoggingEventHandler{}),
		clustering.OptionAliveDelegate(cluster.AliveDefault{}),
	}

	options = append(defaults, options...)
	if c, err = clustering.NewOptions(options...).NewCluster(); err != nil {
		return c, errors.Wrap(err, "failed to join cluster")
	}

	clipeers := peering.Closure(func() ([]string, error) {
		addresses := make([]string, 0, len(t.bootstrap))
		for _, addr := range t.bootstrap {
			addresses = append(addresses, addr.String())
		}

		return addresses, nil
	})

	builtinpeers := peering.Closure(func() ([]string, error) { return []string{}, nil })

	if t.awsBootstrap {
		builtinpeers = t.awsASGPeering()
	}

	joins := clustering.BootstrapOptionJoinStrategy(clustering.MinimumPeers(t.minimumRequiredNodes))
	attempts := clustering.BootstrapOptionAllowRetry(clustering.MaximumAttempts(t.maximumAttempts))
	peerings := clustering.BootstrapOptionPeeringStrategies(
		clipeers,
		builtinpeers,
		snap,
	)

	if err = clustering.Bootstrap(c, peerings, joins, attempts); err != nil {
		return c, errors.Wrap(err, "failed to bootstrap cluster")
	}

	return c, nil
}

func (t *clusterCmd) Snapshot(c clustering.Cluster, fssnapshot peering.File, options ...clustering.SnapshotOption) {
	go clustering.Snapshot(
		c,
		fssnapshot,
		options...,
	)
}

func (t *clusterCmd) Raft(ctx context.Context, conf agent.Config, options ...raftutil.ProtocolOption) (p raftutil.Protocol, err error) {
	var (
		cs    *tls.Config
		snaps raft.SnapshotStore = raft.NewInmemSnapshotStore()
	)

	if cs, err = conf.TLSConfig.BuildServer(); err != nil {
		return p, errors.WithStack(err)
	}

	// if snaps, err = raft.NewFileSnapshotStore(filepath.Join(conf.Root, "raft"), 2, nil); err != nil {
	// 	return p, errors.WithStack(err)
	// }

	defaultOptions := []raftutil.ProtocolOption{
		raftutil.ProtocolOptionEnableSingleNode(t.minimumRequiredNodes == 0),
		raftutil.ProtocolOptionTCPTransport(t.raftNetwork, cs),
		raftutil.ProtocolOptionSnapshotStorage(snaps),
	}

	return raftutil.NewProtocol(
		ctx,
		uint16(t.raftNetwork.Port),
		append(defaultOptions, options...)...,
	)
}

func (t clusterCmd) awsASGPeering() peering.Closure {
	return peering.Closure(func() ([]string, error) {
		instances, err := awsx.AutoscalingPeers()
		if err != nil {
			return []string(nil), err
		}

		result := make([]string, 0, len(instances))
		for _, i := range instances {
			if i.PrivateIpAddress == nil {
				continue
			}

			result = append(result, net.JoinHostPort(*i.PrivateIpAddress, strconv.Itoa(t.swimNetwork.Port)))
		}

		return result, nil
	})
}
