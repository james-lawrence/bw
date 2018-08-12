package main

import (
	"log"
	"net"
	"path/filepath"

	"github.com/alecthomas/kingpin"
	"github.com/davecgh/go-spew/spew"
	"github.com/james-lawrence/bw"
	"github.com/james-lawrence/bw/agent"
	"github.com/james-lawrence/bw/cluster"
	"github.com/james-lawrence/bw/clustering"
	"github.com/james-lawrence/bw/clustering/peering"
	"github.com/james-lawrence/bw/commands/commandutils"
	"github.com/james-lawrence/bw/dns"
)

type cmdDNS struct {
	*global
	config         agent.Config
	configLocation string
	bootstrap      []*net.TCPAddr
	hostedZoneID   string
	region         string
}

func (t *cmdDNS) Configure(parent *kingpin.CmdClause) {
	parent.Flag("agent-name", "name of the node within the network").Default(t.config.Name).StringVar(&t.config.Name)
	parent.Flag("bootstrap", "addresses of the cluster to bootstrap from").PlaceHolder(t.config.SWIMBind.String()).TCPListVar(&t.bootstrap)
	parent.Flag("agent-config", "file containing the agent configuration").Default(t.configLocation).StringVar(&t.configLocation)
	parent.Flag("zone", "hosted zone to insert dns records").Envar("BEARDED_WOOKIE_AWS_DNS_HOSTED_ZONE").StringVar(&t.hostedZoneID)
	parent.Flag("region", "region to insert dns records").Envar("BEARDED_WOOKIE_AWS_DNS_REGION").StringVar(&t.region)
	parent.Action(t.exec)
}

func (t *cmdDNS) exec(ctx *kingpin.ParseContext) error {
	var (
		err    error
		c      clustering.Cluster
		secret []byte
	)

	log.SetPrefix("[BWAWS] ")

	defer t.global.shutdown()

	if err = bw.ExpandAndDecodeFile(t.configLocation, &t.config); err != nil {
		return err
	}

	log.Println("configuration:", spew.Sdump(t.config))

	if err = bw.InitializeDeploymentDirectory(t.config.Root); err != nil {
		return err
	}

	if secret, err = t.config.Hash(); err != nil {
		return err
	}

	local := cluster.NewLocal(
		commandutils.NewClientPeer(),
		cluster.LocalOptionCapability(cluster.NewBitField(cluster.Passive)),
	)

	fssnapshot := peering.File{
		Path: filepath.Join(t.config.Root, "cluster.snapshot"),
	}

	cdialer := commandutils.NewClusterDialer(
		t.config,
		clustering.OptionNodeID(local.Peer.Name),
		clustering.OptionDelegate(local),
		clustering.OptionBindAddress(local.Peer.Ip),
		clustering.OptionBindPort(0),
		clustering.OptionAliveDelegate(cluster.AliveDefault{}),
		clustering.OptionSecret(secret),
	)

	if c, err = commandutils.ClusterJoin(t.global.ctx, t.config, cdialer, fssnapshot, peering.NewStaticTCP(t.bootstrap...)); err != nil {
		return err
	}

	cx := cluster.New(local, c)
	r53dns := dns.MaybeSample(
		dns.NewRoute53(
			t.hostedZoneID,
			t.region,
			dns.Route53OptionCommon(
				dns.OptionTTL(t.config.DNSBind.TTL),
				dns.OptionFQDN(t.config.ServerName),
				dns.OptionMaximumNodes(t.config.MinimumNodes),
			),
		),
	)

	return r53dns(cx)
}
