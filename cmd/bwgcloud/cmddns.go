package main

import (
	"crypto/tls"
	"log"
	"net"

	"cloud.google.com/go/compute/metadata"
	"github.com/alecthomas/kingpin"
	"github.com/davecgh/go-spew/spew"
	"github.com/hashicorp/memberlist"
	"github.com/james-lawrence/bw"
	"github.com/james-lawrence/bw/agent"
	"github.com/james-lawrence/bw/agent/discovery"
	"github.com/james-lawrence/bw/clustering"
	"github.com/james-lawrence/bw/cmd/commandutils"
	"github.com/james-lawrence/bw/daemons"
	"github.com/james-lawrence/bw/dns"
	"github.com/james-lawrence/bw/internal/x/logx"
	"github.com/james-lawrence/bw/internal/x/tlsx"
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

func (t *cmdDNS) exec(ctx *kingpin.ParseContext) (err error) {
	var (
		nodes     []*memberlist.Node
		tlsconfig *tls.Config
	)

	log.SetPrefix("[BWGCLOUD] ")

	defer t.global.shutdown()

	if t.config, err = commandutils.LoadAgentConfig(t.configLocation, t.config); err != nil {
		return err
	}

	log.Println("configuration:", spew.Sdump(t.config))

	if tlsconfig, err = daemons.TLSGenServer(t.config, tlsx.OptionNoClientCert); err != nil {
		return err
	}

	if err = bw.InitializeDeploymentDirectory(t.config.Root); err != nil {
		return err
	}

	d, err := daemons.DefaultDialer(agent.P2PAdddress(t.config.Peer()), tlsconfig)
	if err != nil {
		return err
	}

	if nodes, err = discovery.Snapshot(agent.DiscoveryAddress(t.config.Peer()), d.Defaults()...); err != nil {
		return err
	}

	if len(nodes) == 0 {
		return errors.New("no agents found")
	}

	cx := clustering.NewStatic(nodes...)
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
