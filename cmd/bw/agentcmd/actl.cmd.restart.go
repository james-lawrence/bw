package agentcmd

import (
	"context"
	"log"

	"github.com/james-lawrence/bw"
	"github.com/james-lawrence/bw/agent"
	"github.com/james-lawrence/bw/agent/dialers"
	"github.com/james-lawrence/bw/agentutil"
	"github.com/james-lawrence/bw/cluster"
	"github.com/james-lawrence/bw/clustering"
	"github.com/james-lawrence/bw/cmd/bw/cmdopts"
	"github.com/james-lawrence/bw/deployment"
	"github.com/james-lawrence/bw/internal/systemx"
)

type CmdControlRestart struct {
	controlConnection
	Enabled bool `name:"force" help:"must be specified in order for the command to actual be sent" default:"false"`
}

func (t CmdControlRestart) Run(ctx *cmdopts.Global) error {
	return t.shutdown(ctx.Context, deployment.Or(t.filters()...))
}

func (t CmdControlRestart) shutdown(ctx context.Context, filter deployment.Filter) (err error) {
	var (
		d dialers.Defaults
		c clustering.Rendezvous
	)

	local := &agent.Peer{
		Name: bw.MustGenerateID().String(),
		Ip:   systemx.HostnameOrLocalhost(),
	}

	if d, c, err = t.connect(ctx); err != nil {
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

	return agentutil.Shutdown(ctx, peers, d)
}
