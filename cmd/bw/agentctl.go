package main

import (
	"fmt"
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
	"github.com/james-lawrence/bw/internal/x/systemx"
	"google.golang.org/grpc"
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
		cmd.Arg("environment", "the environment configuration to use").Default(bw.DefaultEnvironmentName).StringVar(&t.environment)
		return cmd
	}

	common(parent.Command("quorum", "print information about the quorum members of the cluster").Default()).Action(t.quorum)
	t.restartCmd(common(parent.Command("restart", "restart all the nodes within the cluster")))

}

func (t *actlCmd) restartCmd(parent *kingpin.CmdClause) *kingpin.CmdClause {
	parent.Flag("name", "regex to match against").RegexpListVar(&t.filteredRegex)
	parent.Flag("ip", "match against the provided IPs").IPListVar(&t.filteredIP)
	parent.Flag("force", "must be specified in order for the command to actual be sent").Default("false").BoolVar(&t.enabled)
	return parent.Action(t.restart)
}

func (t *actlCmd) restart(ctx *kingpin.ParseContext) error {
	filters := make([]deployment.Filter, 0, len(t.filteredRegex))
	for _, n := range t.filteredRegex {
		filters = append(filters, deployment.Named(n))
	}

	for _, n := range t.filteredIP {
		filters = append(filters, deployment.IP(n))
	}

	return t.shutdown(deployment.Or(filters...))
}

func (t *actlCmd) shutdown(filter deployment.Filter) (err error) {
	var (
		d dialers.Defaults
		c clustering.C
	)

	local := cluster.NewLocal(
		&agent.Peer{
			Name: bw.MustGenerateID().String(),
			Ip:   systemx.HostnameOrLocalhost(),
		},
		cluster.LocalOptionCapability(cluster.NewBitField(cluster.Passive)),
	)

	if d, c, err = t.connect(local); err != nil {
		return err
	}
	cx := cluster.New(local, c)

	peers := agentutil.PeerSet(deployment.ApplyFilter(filter, cx.Peers()...))
	if !t.enabled {
		log.Println("force not specified, not executing for the following agents:")
		for _, p := range peers.Peers() {
			log.Println(p.Name, p.Ip)
		}
		return nil
	}

	return agentutil.Shutdown(peers, d)
}

func (t *actlCmd) quorum(ctx *kingpin.ParseContext) (err error) {
	var (
		conn   *grpc.ClientConn
		d      dialers.Defaults
		c      clustering.C
		quorum *agent.InfoResponse
	)

	local := cluster.NewLocal(
		&agent.Peer{
			Name: bw.MustGenerateID().String(),
			Ip:   systemx.HostnameOrLocalhost(),
		},
		cluster.LocalOptionCapability(cluster.NewBitField(cluster.Passive)),
	)

	if d, c, err = t.connect(local); err != nil {
		return err
	}

	fmt.Println("quorum:")
	for idx, p := range agent.QuorumPeers(c) {
		log.Println(idx, p.Name, spew.Sdump(p))
	}

	if conn, err = dialers.NewQuorum(c).Dial(d.Defaults()...); err != nil {
		return err
	}

	if quorum, err = agent.NewQuorumClient(conn).Info(t.global.ctx, &agent.InfoRequest{}); err != nil {
		return err
	}

	peer := func(p *agent.Peer) string {
		if p == nil {
			return "None"
		}

		return fmt.Sprintf("peer %s: %s", p.Name, spew.Sdump(p))
	}

	deployment := func(c *agent.DeployCommand) string {
		if c == nil || c.Archive == nil {
			return "None"
		}

		return fmt.Sprintf("deployment %s - %s - %s", bw.RandomID(c.Archive.DeploymentID), c.Archive.Initiator, c.Command.String())
	}

	fmt.Printf("leader: %s\n", peer(quorum.Leader))
	fmt.Printf("latest: %s\n", deployment(quorum.Deployed))
	fmt.Printf("ongoing: %s\n", deployment(quorum.Deploying))

	return nil
}

func (t actlCmd) connect(local *cluster.Local) (d dialers.Defaults, c clustering.C, err error) {
	var (
		config agent.ConfigClient
	)

	if config, err = commandutils.LoadConfiguration(t.environment); err != nil {
		return d, c, err
	}

	log.Println("configuration:", spew.Sdump(config))

	coptions := []daemons.ConnectOption{
		daemons.ConnectOptionClustering(
			clustering.OptionNodeID(local.Peer.Name),
			clustering.OptionBindAddress(local.Peer.Ip),
			clustering.OptionEventDelegate(cluster.LoggingEventHandler{}),
			clustering.OptionAliveDelegate(cluster.AliveDefault{}),
		),
	}

	if d, c, err = daemons.Connect(config, coptions...); err != nil {
		return d, c, err
	}

	log.Println("connected to cluster")
	debugx.Printf("configuration:\n%#v\n", config)
	return d, c, nil
}
