package main

import (
	"context"
	"crypto/tls"
	"log"
	"net"
	"os"
	"path/filepath"
	"strconv"

	"github.com/alecthomas/kingpin"
	"github.com/hashicorp/memberlist"
	"github.com/hashicorp/raft"
	raftboltdb "github.com/hashicorp/raft-boltdb"
	"github.com/james-lawrence/bw"
	"github.com/james-lawrence/bw/agent"
	"github.com/james-lawrence/bw/agent/dialers"
	"github.com/james-lawrence/bw/agent/discovery"
	"github.com/james-lawrence/bw/certificatecache"
	"github.com/james-lawrence/bw/clustering"
	"github.com/james-lawrence/bw/clustering/peering"
	"github.com/james-lawrence/bw/clustering/raftutil"
	"github.com/james-lawrence/bw/cmd/commandutils"
	"github.com/james-lawrence/bw/internal/x/envx"
	"github.com/james-lawrence/bw/internal/x/errorsx"
	"github.com/james-lawrence/bw/internal/x/tlsx"
	"github.com/james-lawrence/bw/notary"
	"github.com/pkg/errors"
	"google.golang.org/grpc"
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
	parent.Flag("cluster-dns-enable", "enable dns bootstrap").Default("false").Envar(
		bw.EnvAgentClusterEnableDNS,
	).BoolVar(&t.dnsEnabled)
	parent.Flag("cluster-aws-enable", "enable/disable aws autoscale group bootstrap").Default("false").BoolVar(&t.awsEnabled)
	parent.Flag("cluster-gcloud-enable", "enable/disable gcloud target pools bootstrap").Default("false").
		Envar(bw.EnvAgentClusterEnableGoogleCloudPool).
		BoolVar(&t.gcloudEnabled)
}

func (t *clusterCmd) Join(ctx context.Context, conf agent.Config, c clustering.Joiner, snap peering.File) (err error) {
	var (
		p2ppeers    clustering.Source
		clipeers    clustering.Source = peering.NewStaticTCP(t.bootstrap...)
		awspeers    clustering.Source = peering.NewStaticTCP()
		gcloudpeers clustering.Source = peering.NewStaticTCP()
		dnspeers    clustering.Source = peering.NewStaticTCP()
	)

	if p2ppeers, err = p2ppeering(conf); err != nil {
		log.Println("WARNING: P2P discovery disabled", err)
		p2ppeers = peering.NewStaticTCP()
	}

	if t.dnsEnabled {
		log.Println("dns peering enabled")
		dnspeers = peering.NewDNS(t.config.P2PBind.Port, append(t.config.DNSBootstrap, t.config.ServerName)...)
	}

	if t.awsEnabled {
		log.Println("aws autoscale groups peering enabled")
		awspeers = peering.AWSAutoscaling{
			Port:               conf.P2PBind.Port,
			SupplimentalGroups: conf.AWSBootstrap.AutoscalingGroups,
		}
	}

	if t.gcloudEnabled {
		log.Println("gcloud target pool peering enabled")
		gcloudpeers = peering.GCloudTargetPool{
			Port:    conf.P2PBind.Port,
			Maximum: conf.MinimumNodes,
		}
	}

	return commandutils.ClusterJoin(ctx, conf, c, clipeers, p2ppeers, awspeers, gcloudpeers, snap, dnspeers)
}

func (t *clusterCmd) Snapshot(c clustering.Rendezvous, fssnapshot peering.File, options ...clustering.SnapshotOption) {
	go clustering.Snapshot(
		c,
		fssnapshot,
		options...,
	)
}

func (t *clusterCmd) Raft(ctx context.Context, conf agent.Config, node *memberlist.Node, eq *grpc.ClientConn, options ...raftutil.ProtocolOption) (p raftutil.Protocol, err error) {
	var (
		dir = filepath.Join(conf.Root, "raft.d")
	)

	if err = os.MkdirAll(dir, 0700); err != nil {
		return p, err
	}

	defaultOptions := []raftutil.ProtocolOption{
		raftutil.ProtocolOptionQuorumMinimum(conf.MinimumNodes),
		raftutil.ProtocolOptionEnableSingleNode(conf.MinimumNodes <= 1),
		raftutil.ProtocolOptionPassiveReset(func() (s raftutil.Storage, ss raft.SnapshotStore, err error) {
			if err = errorsx.Compact(os.RemoveAll(dir), os.MkdirAll(dir, 0700)); err != nil {
				return s, ss, errors.WithStack(err)
			}

			if s, err = raftStore(conf); err != nil {
				return s, ss, errors.WithStack(err)
			}

			if ss, err = raft.NewFileSnapshotStore(dir, 5, os.Stderr); err != nil {
				return s, ss, errors.WithStack(err)
			}

			return s, ss, nil
		}),
	}

	return raftutil.NewProtocol(
		ctx,
		node,
		eq,
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

type p2p struct {
	address string
	d       dialers.Defaults
}

func (t p2p) Peers(ctx context.Context) (results []string, err error) {
	var (
		nodes []*memberlist.Node
	)

	if nodes, err = discovery.Snapshot(t.address, t.d.Defaults()...); err != nil {
		return nil, err
	}

	for _, n := range nodes {
		results = append(results, n.Address())
	}

	return results, nil
}

func p2ppeering(c agent.Config) (s clustering.Source, err error) {
	var (
		tlsconfig *tls.Config
		ss        notary.Signer
		d         dialers.Defaults
		address   = net.JoinHostPort(c.ServerName, envx.String(strconv.Itoa(c.P2PBind.Port), bw.EnvAgentClusterP2PDiscoveryPort))
	)

	if ss, err = notary.NewAgentSigner(c.Root); err != nil {
		return nil, err
	}

	if tlsconfig, err = certificatecache.TLSGenServer(c, tlsx.OptionNoClientCert); err != nil {
		return nil, err
	}

	d, err = dialers.DefaultDialer(address, tlsx.NewDialer(tlsconfig), grpc.WithPerRPCCredentials(ss))
	if err != nil {
		return nil, err
	}

	return p2p{
		address: agent.DiscoveryP2PAddress(address),
		d:       d,
	}, nil
}
