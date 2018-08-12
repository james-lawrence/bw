package main

import (
	"log"
	"net"
	"path/filepath"

	"cloud.google.com/go/compute/metadata"
	"github.com/alecthomas/kingpin"
	"github.com/davecgh/go-spew/spew"
	"github.com/james-lawrence/bw"
	"github.com/james-lawrence/bw/agent"
	"github.com/james-lawrence/bw/cluster"
	"github.com/james-lawrence/bw/clustering"
	"github.com/james-lawrence/bw/clustering/peering"
	"github.com/james-lawrence/bw/commands/commandutils"
	"github.com/james-lawrence/bw/dns"
	"github.com/james-lawrence/bw/x/logx"
	"github.com/pkg/errors"
)

type cmdDNS struct {
	*global
	config         agent.Config
	configLocation string
	bootstrap      []*net.TCPAddr
	zoneID         string
	projectID      string
}

func (t *cmdDNS) Configure(parent *kingpin.CmdClause) {
	var (
		err error
	)

	if metadata.OnGCE() {
		if t.projectID, err = metadata.ProjectID(); err != nil {
			logx.MaybeLog(errors.Wrap(err, "failed to retrieve project ID from metadata service"))
		}
	}

	parent.Flag("agent-name", "name of the node within the network").Default(t.config.Name).StringVar(&t.config.Name)
	parent.Flag("bootstrap", "addresses of the cluster to bootstrap from").PlaceHolder(t.config.SWIMBind.String()).TCPListVar(&t.bootstrap)
	parent.Flag("agent-config", "file containing the agent configuration").Default(t.configLocation).StringVar(&t.configLocation)
	parent.Flag("projectID", "gcloud project id usually pulled from metadata automatically").Envar("BEARDED_WOOKIE_GCLOUD_PROJECT_ID").PlaceHolder(t.projectID).Default(t.projectID).StringVar(&t.projectID)
	parent.Flag("zone", "dns zone where changes will be applied").Envar("BEARDED_WOOKIE_GCLOUD_DNS_ZONE").StringVar(&t.zoneID)

	parent.Action(t.exec)
}

func (t *cmdDNS) exec(ctx *kingpin.ParseContext) error {
	var (
		err    error
		c      clustering.Cluster
		secret []byte
	)

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
	return dns.NewGoogleCloudDNS(
		t.projectID,
		t.zoneID,
		dns.GCloudDNSOptionCommon(
			dns.OptionTTL(t.config.DNSBind.TTL),
			dns.OptionFQDN(t.config.ServerName),
			dns.OptionMaximumNodes(t.config.MinimumNodes),
		),
	).Sample(cx)
}
