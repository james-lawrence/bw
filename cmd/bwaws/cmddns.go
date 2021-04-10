package main

import (
	"crypto/tls"
	"errors"
	"log"
	"net"

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
	"github.com/james-lawrence/bw/internal/x/tlsx"
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

func (t *cmdDNS) exec(ctx *kingpin.ParseContext) (err error) {
	var (
		nodes     []*memberlist.Node
		tlsconfig *tls.Config
	)

	log.SetPrefix("[BWAWS] ")

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
