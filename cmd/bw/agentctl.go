package main

import (
	"log"
	"net"
	"regexp"

	"github.com/alecthomas/kingpin"
	"github.com/davecgh/go-spew/spew"
	"github.com/james-lawrence/bw"
	"github.com/james-lawrence/bw/agent"
	"github.com/james-lawrence/bw/agent/dialers"
	"github.com/james-lawrence/bw/agentutil"
	"github.com/james-lawrence/bw/cluster"
	"github.com/james-lawrence/bw/clustering"
	"github.com/james-lawrence/bw/cmd/commandutils"
	"github.com/james-lawrence/bw/daemons"
	"github.com/james-lawrence/bw/deployment"
	"github.com/james-lawrence/bw/internal/x/debugx"
	"github.com/james-lawrence/bw/internal/x/logx"
	"github.com/james-lawrence/bw/internal/x/systemx"
	"github.com/pkg/errors"
)

type actlCmd struct {
	global        *global
	enabled       bool
	environment   string
	filteredIP    []net.IP
	filteredRegex []*regexp.Regexp
}

func (t *actlCmd) configure(parent *kingpin.CmdClause) {
	common := func(cmd *kingpin.CmdClause) *kingpin.CmdClause {
		cmd.Flag("force", "must be specified in order for the command to actual be sent").Default("false").BoolVar(&t.enabled)
		cmd.Arg("environment", "the environment configuration to use").Default(bw.DefaultEnvironmentName).StringVar(&t.environment)
		return cmd
	}

	t.actlCmd(common(parent.Command("all", "restart all the nodes within the cluster").Default()))
	t.filteredCmd(common(parent.Command("filtered", "restart all the nodes that match one of the provided filters")))
}

func (t *actlCmd) actlCmd(parent *kingpin.CmdClause) *kingpin.CmdClause {
	return parent.Action(t.all)
}

func (t *actlCmd) filteredCmd(parent *kingpin.CmdClause) *kingpin.CmdClause {
	parent.Flag("name", "regex to match against").RegexpListVar(&t.filteredRegex)
	parent.Flag("ip", "match against the provided IPs").IPListVar(&t.filteredIP)
	return parent.Action(t.filtered)
}

func (t *actlCmd) filtered(ctx *kingpin.ParseContext) error {
	filters := make([]deployment.Filter, 0, len(t.filteredRegex))
	for _, n := range t.filteredRegex {
		filters = append(filters, deployment.Named(n))
	}

	for _, n := range t.filteredIP {
		filters = append(filters, deployment.IP(n))
	}

	return t.shutdown(deployment.Or(filters...))
}

func (t *actlCmd) all(ctx *kingpin.ParseContext) error {
	return t.shutdown(deployment.AlwaysMatch)
}

func (t *actlCmd) shutdown(filter deployment.Filter) (err error) {
	var (
		client agent.Client
		config agent.ConfigClient
		dialer dialers.Quorum
		c      clustering.Cluster
	)

	if config, err = commandutils.LoadConfiguration(t.environment); err != nil {
		return err
	}

	log.Println("configuration:", spew.Sdump(config))

	local := cluster.NewLocal(
		agent.Peer{
			Name: bw.MustGenerateID().String(),
			Ip:   systemx.HostnameOrLocalhost(),
		},
		cluster.LocalOptionCapability(cluster.NewBitField(cluster.Passive)),
	)

	coptions := []daemons.ConnectOption{
		daemons.ConnectOptionClustering(
			clustering.OptionDelegate(local),
			clustering.OptionNodeID(local.Peer.Name),
			clustering.OptionBindAddress(local.Peer.Ip),
			clustering.OptionEventDelegate(cluster.LoggingEventHandler{}),
			clustering.OptionAliveDelegate(cluster.AliveDefault{}),
		),
	}

	if client, dialer, c, err = daemons.Connect(config, coptions...); err != nil {
		return err
	}

	logx.MaybeLog(errors.WithMessage(client.Close(), "failed to close client"))
	log.Println("connected to cluster")
	debugx.Printf("configuration:\n%#v\n", config)

	cx := cluster.New(local, c)

	peers := agentutil.PeerSet(deployment.ApplyFilter(filter, cx.Peers()...))
	if !t.enabled {
		log.Println("force not specified, not executing for the following agents:")
		for _, p := range peers.Peers() {
			log.Println(p.Name, p.Ip)
		}
		return nil
	}

	return agentutil.Shutdown(peers, agent.NewDialer(dialer.Defaults()...))
}
