package main

import (
	"context"
	"crypto/tls"
	"log"
	"net"
	"os"
	"path/filepath"

	"github.com/alecthomas/kingpin"
	"github.com/hashicorp/raft"
	raftboltdb "github.com/hashicorp/raft-boltdb"
	"github.com/james-lawrence/bw"
	"github.com/james-lawrence/bw/agent"
	"github.com/james-lawrence/bw/clustering"
	"github.com/james-lawrence/bw/clustering/peering"
	"github.com/james-lawrence/bw/clustering/raftutil"
	"github.com/james-lawrence/bw/cmd/commandutils"
	"github.com/james-lawrence/bw/daemons"
	"github.com/pkg/errors"
)

type clusterCmd struct {
	config                                *agent.Config
	bootstrap                             []*net.TCPAddr
	dnsEnabled, awsEnabled, gcloudEnabled bool
}

func (t *clusterCmd) configure(parent *kingpin.CmdClause, config *agent.Config) {
	t.config = config
	parent.Flag("cluster", "addresses of the cluster to bootstrap from").PlaceHolder(
		t.config.SWIMBind.String(),
	).Envar(
		bw.EnvAgentClusterBootstrap,
	).TCPListVar(&t.bootstrap)
	parent.Flag(
		"cluster-bind",
		"address for the swim protocol (cluster membership) to bind to",
	).PlaceHolder(
		t.config.SWIMBind.String(),
	).Envar(
		bw.EnvAgentSWIMBind,
	).TCPVar(&t.config.SWIMBind)
	parent.Flag(
		"cluster-bind-raft",
		"address for the raft protocol to bind to",
	).PlaceHolder(
		t.config.RaftBind.String(),
	).Envar(
		bw.EnvAgentRAFTBind,
	).TCPVar(&t.config.RaftBind)

	parent.Flag("cluster-dns-enable", "enable dns bootstrap").BoolVar(&t.dnsEnabled)
	parent.Flag("cluster-aws-enable", "enable/disable aws autoscale group bootstrap").Default("true").BoolVar(&t.awsEnabled)
	parent.Flag("cluster-gcloud-enable", "enable/disable gcloud target pools bootstrap").Default("true").BoolVar(&t.gcloudEnabled)
}

func (t *clusterCmd) Join(ctx context.Context, conf agent.Config, d clustering.Dialer, snap peering.File) (clustering.Cluster, error) {
	var (
		awspeers           clustering.Source = peering.NewStaticTCP()
		gcloudpeers        clustering.Source = peering.NewStaticTCP()
		dnspeers           clustering.Source = peering.NewDNS(t.config.SWIMBind.Port)
		dnspeersDeprecated clustering.Source = peering.NewStaticTCP()
		clipeers           clustering.Source = peering.NewStaticTCP(t.bootstrap...)
	)

	if t.dnsEnabled {
		log.Println("dns peering enabled")
		dnspeers = peering.NewDNS(t.config.SWIMBind.Port, append(t.config.DNSBootstrap, t.config.ServerName)...)
		dnspeersDeprecated = peering.NewDNS(2001, append(t.config.DNSBootstrap, t.config.ServerName)...)
	}

	if t.awsEnabled {
		log.Println("aws autoscale groups peering enabled")
		awspeers = peering.AWSAutoscaling{
			Port:               conf.SWIMBind.Port,
			SupplimentalGroups: conf.AWSBootstrap.AutoscalingGroups,
		}
	}

	if t.gcloudEnabled {
		log.Println("gcloud target pool peering enabled")
		gcloudpeers = peering.GCloudTargetPool{
			Port:    conf.SWIMBind.Port,
			Maximum: conf.MinimumNodes,
		}
	}

	return commandutils.ClusterJoin(ctx, conf, d, clipeers, dnspeers, dnspeersDeprecated, awspeers, gcloudpeers, snap)
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
		s  *raftboltdb.BoltStore
		ss raft.SnapshotStore
	)

	if err = os.MkdirAll(filepath.Join(conf.Root, "raft.d"), 0700); err != nil {
		return p, err
	}

	if s, err = raftStore(conf); err != nil {
		return p, errors.WithStack(err)
	}

	if cs, err = daemons.TLSGenServer(conf); err != nil {
		return p, errors.WithStack(err)
	}

	if ss, err = raft.NewFileSnapshotStore(filepath.Join(conf.Root, "raft.d"), 5, os.Stderr); err != nil {
		return p, errors.WithStack(err)
	}

	defaultOptions := []raftutil.ProtocolOption{
		raftutil.ProtocolOptionEnableSingleNode(conf.MinimumNodes == 0),
		raftutil.ProtocolOptionTCPTransport(conf.RaftBind, cs),
		raftutil.ProtocolOptionStorage(s),
		raftutil.ProtocolOptionSnapshotStorage(ss),
	}

	return raftutil.NewProtocol(
		ctx,
		sq,
		append(defaultOptions, options...)...,
	)
}

func raftStore(c agent.Config) (*raftboltdb.BoltStore, error) {
	return raftStoreFilepath(filepath.Join(c.Root, "raft.d", "state.bin"))
}

func raftStoreFilepath(p string) (*raftboltdb.BoltStore, error) {
	sopts := raftboltdb.Options{
		Path: p,
	}
	return raftboltdb.New(sopts)
}