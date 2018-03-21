package commandutils

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
	"github.com/pkg/errors"
)

// ClusterCmd utility to join a swim cluster.
type ClusterCmd struct {
	Config                 *agent.Config
	bootstrap              []*net.TCPAddr
	dnsEnabled, awsEnabled bool
}

// Configure ...
func (t *ClusterCmd) Configure(parent *kingpin.CmdClause, config *agent.Config) {
	t.Config = config
	parent.Flag("cluster", "addresses of the cluster to bootstrap from").PlaceHolder(t.Config.SWIMBind.String()).TCPListVar(&t.bootstrap)
	parent.Flag("cluster-bind", "address for the swim protocol (cluster membership) to bind to").PlaceHolder(t.Config.SWIMBind.String()).TCPVar(&t.Config.SWIMBind)
	parent.Flag("cluster-bind-raft", "address for the raft protocol to bind to").PlaceHolder(t.Config.RaftBind.String()).TCPVar(&t.Config.RaftBind)
	parent.Flag("cluster-dns-enable", "enable dns bootstrap").BoolVar(&t.dnsEnabled)
	parent.Flag("cluster-aws-enable", "enable aws autoscale group bootstrap").BoolVar(&t.awsEnabled)
}

// Join ...
func (t *ClusterCmd) Join(ctx context.Context, conf agent.Config, d clustering.Dialer, snap peering.File) (clustering.Cluster, error) {
	var (
		awspeers clustering.Source = peering.NewStaticTCP()
		dnspeers clustering.Source = peering.NewDNS(t.Config.SWIMBind.Port)
		clipeers clustering.Source = peering.NewStaticTCP(t.bootstrap...)
	)

	if t.dnsEnabled {
		log.Println("dns peering enabled")
		dnspeers = peering.NewDNS(t.Config.SWIMBind.Port, append(t.Config.DNSBootstrap, t.Config.ServerName)...)
	}

	if t.awsEnabled {
		log.Println("aws autoscale groups peering enabled")
		awspeers = peering.AWSAutoscaling{
			Port:               conf.SWIMBind.Port,
			SupplimentalGroups: conf.AWSBootstrap.AutoscalingGroups,
		}
	}

	return ClusterJoin(ctx, conf, d, clipeers, dnspeers, awspeers, snap)
}

// Snapshot ...
func (t *ClusterCmd) Snapshot(c clustering.Cluster, fssnapshot peering.File, options ...clustering.SnapshotOption) {
	go clustering.Snapshot(
		c,
		fssnapshot,
		options...,
	)
}

// Raft ...
func (t *ClusterCmd) Raft(ctx context.Context, conf agent.Config, sq raftutil.BacklogQueueWorker, options ...raftutil.ProtocolOption) (p raftutil.Protocol, err error) {
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
