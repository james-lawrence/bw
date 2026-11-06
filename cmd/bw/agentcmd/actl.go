package agentcmd

import (
	"context"
	"log"
	"net"
	"regexp"

	"github.com/james-lawrence/bw"
	"github.com/james-lawrence/bw/agent"
	"github.com/james-lawrence/bw/agent/dialers"
	"github.com/james-lawrence/bw/clustering"
	"github.com/james-lawrence/bw/clustering/rendezvous"
	"github.com/james-lawrence/bw/cmd/bw/cmdopts"
	"github.com/james-lawrence/bw/cmd/commandutils"
	"github.com/james-lawrence/bw/daemons"
	"github.com/james-lawrence/bw/deployment"
	"github.com/james-lawrence/bw/internal/envx"
	"github.com/james-lawrence/bw/notary"
	"github.com/james-lawrence/bw/vcsinfo"
	"google.golang.org/grpc"
)

type CmdControl struct {
	Restart CmdControlRestart `cmd:"" help:"restart all the nodes within the cluster"`
	Quorum  CmdControlQuorum  `cmd:"" help:"print information about the quorum members of the cluster"`
	Watch   CmdControlWatch   `cmd:"" help:"watch cluster activity"`
	Nodes   CmdControlNodes   `cmd:"" help:"retrieve nodes within the cluster"`
	Logs    CmdControlLogs    `cmd:"" help:"log retrieval for the latest deployment"`
	Check   CmdControlCheck   `cmd:"" help:"check connectivity with the discovery service"`
}

type controlConnection struct {
	cmdopts.BeardedWookieEnv
	Names    []*regexp.Regexp `name:"name" help:"regex to match names against"`
	IPs      []net.IP         `name:"ip" help:"match against the provided IP addresses"`
	Canary   bool             `name:"canary" help:"deploy to the canary server" default:"false"`
	Insecure bool             `name:"insecure" help:"disable tls verification"`
}

func (t controlConnection) connect(ctx context.Context) (d dialers.Defaults, c clustering.Rendezvous, err error) {
	var (
		config agent.ConfigClient
		ss     notary.Signer
	)

	if config, err = commandutils.LoadConfiguration(ctx, t.Environment, agent.CCOptionInsecure(t.Insecure)); err != nil {
		return d, c, err
	}

	if ss, err = notary.NewAutoSigner(vcsinfo.CurrentUserDisplay(config.WorkDir())); err != nil {
		return d, c, err
	}

	if d, c, err = daemons.Connect(ctx, config, ss, grpc.WithPerRPCCredentials(ss)); err != nil {
		return d, c, err
	}

	if envx.Boolean(false, bw.EnvLogsVerbose) {
		log.Println("connected to cluster")
	}

	return d, c, nil
}

func (t controlConnection) filters(additional ...deployment.Filter) []deployment.Filter {
	filters := make([]deployment.Filter, 0, 1+len(t.Names)+len(t.IPs))
	for _, n := range t.Names {
		filters = append(filters, deployment.Named(n))
	}

	for _, n := range t.IPs {
		filters = append(filters, deployment.IP(n))
	}

	for _, a := range additional {
		if a == nil {
			continue
		}

		filters = append(filters, a)
	}

	if len(filters) == 0 {
		filters = append(filters, deployment.AlwaysMatch)
	}

	return filters
}

func (t controlConnection) canary(c clustering.Rendezvous) deployment.Filter {
	if t.Canary {
		return deployment.IP(c.Get(rendezvous.Auto()).Addr)
	}

	return nil
}

func (t controlConnection) filtered(c clustering.Rendezvous) clustering.Rendezvous {
	filters := t.filters(t.canary(c))
	return clustering.NewCached(func(ctx context.Context) clustering.Rendezvous {
		peers := agent.NodesToPeers(c.Members()...)
		return clustering.NewStatic(agent.PeersToNodes(
			deployment.ApplyFilter(deployment.Or(filters...), peers...)...,
		)...)
	})
}
