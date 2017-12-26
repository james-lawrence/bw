package main

import (
	"context"
	"crypto/tls"
	"log"
	"net"

	"github.com/james-lawrence/bw/agent"
	"github.com/james-lawrence/bw/cluster"
	"github.com/james-lawrence/bw/clustering"
	"github.com/james-lawrence/bw/clustering/peering"
	"github.com/james-lawrence/bw/clustering/raftutil"

	"github.com/alecthomas/kingpin"
	"github.com/pkg/errors"
)

type clusterCmd struct {
	config               *agent.Config
	bootstrap            []*net.TCPAddr
	AWSAutoscalingGroups []string
}

func (t *clusterCmd) configure(parent *kingpin.CmdClause, config *agent.Config) {
	t.config = config
	parent.Flag("cluster", "addresses of the cluster to bootstrap from").PlaceHolder(t.config.SWIMBind.String()).TCPListVar(&t.bootstrap)
	parent.Flag("cluster-bind", "address for the swim protocol (cluster membership) to bind to").PlaceHolder(t.config.SWIMBind.String()).TCPVar(&t.config.SWIMBind)
	parent.Flag("cluster-bind-raft", "address for the raft protocol to bind to").PlaceHolder(t.config.RaftBind.String()).TCPVar(&t.config.RaftBind)
	parent.Flag("cluster-asg", "autoscaling groups to check, only useful if you have multiple autoscaling groups").StringsVar(&t.AWSAutoscalingGroups)
}

func (t *clusterCmd) Join(conf agent.Config, snap peering.File, options ...clustering.Option) (clustering.Cluster, error) {
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
		clustering.OptionBindAddress(conf.SWIMBind.IP.String()),
		clustering.OptionBindPort(conf.SWIMBind.Port),
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

	joins := clustering.BootstrapOptionJoinStrategy(clustering.MinimumPeers(conf.MinimumPeers))
	attempts := clustering.BootstrapOptionAllowRetry(clustering.MaximumAttempts(conf.BootstrapAttempts))
	peerings := clustering.BootstrapOptionPeeringStrategies(
		snap,
		clipeers,
		peering.AWSAutoscaling{
			Port:               conf.SWIMBind.Port,
			SupplimentalGroups: t.AWSAutoscalingGroups,
		},
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
		cs *tls.Config
	)

	if cs, err = conf.TLSConfig.BuildServer(); err != nil {
		return p, errors.WithStack(err)
	}

	defaultOptions := []raftutil.ProtocolOption{
		raftutil.ProtocolOptionEnableSingleNode(conf.MinimumPeers == 0),
		raftutil.ProtocolOptionTCPTransport(conf.RaftBind, cs),
	}

	return raftutil.NewProtocol(
		ctx,
		uint16(conf.RaftBind.Port),
		append(defaultOptions, options...)...,
	)
}
