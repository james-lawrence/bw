package main

import (
	"context"
	"crypto/tls"
	"log"
	"net"

	"github.com/alecthomas/kingpin"
	"github.com/james-lawrence/bw/agent"
	"github.com/james-lawrence/bw/clustering"
	"github.com/james-lawrence/bw/clustering/peering"
	"github.com/james-lawrence/bw/clustering/raftutil"
	"github.com/james-lawrence/bw/cmd/commandutils"
	"github.com/pkg/errors"
)

type clusterCmd struct {
	config                 *agent.Config
	bootstrap              []*net.TCPAddr
	dnsEnabled, awsEnabled bool
}

func (t *clusterCmd) configure(parent *kingpin.CmdClause, config *agent.Config) {
	t.config = config
	parent.Flag("cluster", "addresses of the cluster to bootstrap from").PlaceHolder(t.config.SWIMBind.String()).TCPListVar(&t.bootstrap)
	parent.Flag("cluster-bind", "address for the swim protocol (cluster membership) to bind to").PlaceHolder(t.config.SWIMBind.String()).TCPVar(&t.config.SWIMBind)
	parent.Flag("cluster-bind-raft", "address for the raft protocol to bind to").PlaceHolder(t.config.RaftBind.String()).TCPVar(&t.config.RaftBind)
	parent.Flag("cluster-dns-enable", "enable dns bootstrap").BoolVar(&t.dnsEnabled)
	parent.Flag("cluster-aws-enable", "enable aws autoscale group bootstrap").BoolVar(&t.awsEnabled)
}

func (t *clusterCmd) Join(ctx context.Context, conf agent.Config, d clustering.Dialer, snap peering.File) (clustering.Cluster, error) {
	var (
		awspeers clustering.Source = peering.NewStaticTCP()
		dnspeers clustering.Source = peering.NewDNS(t.config.SWIMBind.Port)
		clipeers clustering.Source = peering.NewStaticTCP(t.bootstrap...)
	)

	if t.dnsEnabled {
		log.Println("dns peering enabled")
		dnspeers = peering.NewDNS(t.config.SWIMBind.Port, append(t.config.DNSBootstrap, t.config.ServerName)...)
	}

	if t.awsEnabled {
		log.Println("aws autoscale groups peering enabled")
		awspeers = peering.AWSAutoscaling{
			Port:               conf.SWIMBind.Port,
			SupplimentalGroups: conf.AWSBootstrap.AutoscalingGroups,
		}
	}

	return commandutils.ClusterJoin(ctx, conf, d, clipeers, dnspeers, awspeers, snap)
}

func (t *clusterCmd) Snapshot(c clustering.Cluster, fssnapshot peering.File, options ...clustering.SnapshotOption) {
	go clustering.Snapshot(
		c,
		fssnapshot,
		options...,
	)
}

func (t *clusterCmd) Raft(ctx context.Context, conf agent.Config, sq raftutil.BacklogQueueWorker, options ...raftutil.ProtocolOption) (p raftutil.Protocol, err error) {
	var (
		cs *tls.Config
	)

	if cs, err = conf.BuildServer(); err != nil {
		return p, errors.WithStack(err)
	}

	defaultOptions := []raftutil.ProtocolOption{
		raftutil.ProtocolOptionEnableSingleNode(conf.MinimumNodes == 0),
		raftutil.ProtocolOptionTCPTransport(conf.RaftBind, cs),
	}

	return raftutil.NewProtocol(
		ctx,
		sq,
		append(defaultOptions, options...)...,
	)
}
