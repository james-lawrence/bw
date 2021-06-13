package main

import (
	"log"
	"net"
	"regexp"

	"github.com/alecthomas/kingpin"
	"github.com/james-lawrence/bw"
	"github.com/james-lawrence/bw/agent"
	"github.com/james-lawrence/bw/agent/dialers"
	"github.com/james-lawrence/bw/agentutil"
	"github.com/james-lawrence/bw/cluster"
	"github.com/james-lawrence/bw/clustering"
	"github.com/james-lawrence/bw/cmd/commandutils"
	"github.com/james-lawrence/bw/daemons"
	"github.com/james-lawrence/bw/deployment"
	"github.com/james-lawrence/bw/internal/x/envx"
	"github.com/james-lawrence/bw/internal/x/systemx"
	"github.com/james-lawrence/bw/notary"
	"github.com/james-lawrence/bw/uxterm"
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
	filters := make([]deployment.Filter, 0, 1+len(t.filteredRegex)+len(t.filteredIP))
	for _, n := range t.filteredRegex {
		filters = append(filters, deployment.Named(n))
	}

	for _, n := range t.filteredIP {
		filters = append(filters, deployment.IP(n))
	}

	if len(filters) == 0 {
		filters = append(filters, deployment.AlwaysMatch)
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
		ss     notary.Signer
		conn   *grpc.ClientConn
		d      dialers.Defaults
		c      clustering.C
		quorum *agent.InfoResponse
	)

	if ss, err = notary.NewAutoSigner(bw.DisplayName()); err != nil {
		return err
	}

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

	if conn, err = dialers.NewQuorum(c).Dial(d.Defaults(grpc.WithPerRPCCredentials(ss))...); err != nil {
		return err
	}

	if quorum, err = agent.NewQuorumClient(conn).Info(t.global.ctx, &agent.InfoRequest{}); err != nil {
		return err
	}

	if err = uxterm.PrintQuorum(quorum); err != nil {
		return err
	}

	return nil
}

func (t actlCmd) connect(local *cluster.Local) (d dialers.Defaults, c clustering.C, err error) {
	var (
		config agent.ConfigClient
		ss     notary.Signer
	)

	if config, err = commandutils.LoadConfiguration(t.environment); err != nil {
		return d, c, err
	}

	if ss, err = notary.NewAutoSigner(bw.DisplayName()); err != nil {
		return d, c, err
	}

	if d, c, err = daemons.Connect(config, ss); err != nil {
		return d, c, err
	}

	if envx.Boolean(false, bw.EnvLogsVerbose) {
		log.Println("connected to cluster")
	}

	return d, c, nil
}
