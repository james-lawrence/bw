package agentcmd

import (
	"log"
	"net"
	"regexp"

	"github.com/james-lawrence/bw"
	"github.com/james-lawrence/bw/agent"
	"github.com/james-lawrence/bw/agent/dialers"
	"github.com/james-lawrence/bw/agentutil"
	"github.com/james-lawrence/bw/cluster"
	"github.com/james-lawrence/bw/clustering"
	"github.com/james-lawrence/bw/cmd/bwc/cmdopts"
	"github.com/james-lawrence/bw/cmd/commandutils"
	"github.com/james-lawrence/bw/daemons"
	"github.com/james-lawrence/bw/deployment"
	"github.com/james-lawrence/bw/internal/x/envx"
	"github.com/james-lawrence/bw/internal/x/systemx"
	"github.com/james-lawrence/bw/notary"
	"github.com/james-lawrence/bw/uxterm"
	"google.golang.org/grpc"
)

type CmdControl struct {
	Restart CmdControlRestart `cmd:"" help:"restart all the nodes within the cluster"`
	Quorum  CmdControlQuorum  `cmd:"" help:"print information about the quorum members of the cluster"`
}

type controlConnection struct {
	cmdopts.BeardedWookieEnv
}

func (t controlConnection) connect(local *cluster.Local) (d dialers.Defaults, c clustering.C, err error) {
	var (
		config agent.ConfigClient
		ss     notary.Signer
	)

	if config, err = commandutils.LoadConfiguration(t.Environment); err != nil {
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

type CmdControlRestart struct {
	controlConnection
	Names   []*regexp.Regexp `name:"name" help:"regex to match names against"`
	IPs     []net.IP         `name:"ip" help:"match against the provided IP addresses"`
	Enabled bool             `name:"force" help:"must be specified in order for the command to actual be sent" default:"false"`
}

func (t CmdControlRestart) Run() error {
	filters := make([]deployment.Filter, 0, 1+len(t.Names)+len(t.IPs))
	for _, n := range t.Names {
		filters = append(filters, deployment.Named(n))
	}

	for _, n := range t.IPs {
		filters = append(filters, deployment.IP(n))
	}

	if len(filters) == 0 {
		filters = append(filters, deployment.AlwaysMatch)
	}

	return t.shutdown(deployment.Or(filters...))
}

func (t CmdControlRestart) shutdown(filter deployment.Filter) (err error) {
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
	if !t.Enabled {
		log.Println("force not specified, not executing for the following agents:")
		for _, p := range peers.Peers() {
			log.Println(p.Name, p.Ip)
		}
		return nil
	}

	return agentutil.Shutdown(peers, d)
}

type CmdControlQuorum struct {
	controlConnection
}

func (t CmdControlQuorum) Run(ctx *cmdopts.Global) (err error) {
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

	if quorum, err = agent.NewQuorumClient(conn).Info(ctx.Context, &agent.InfoRequest{}); err != nil {
		return err
	}

	if err = uxterm.PrintQuorum(quorum); err != nil {
		return err
	}

	return nil
}
