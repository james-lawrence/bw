package agentcmd

import (
	"github.com/james-lawrence/bw/agent"
	"github.com/james-lawrence/bw/agent/dialers"
	"github.com/james-lawrence/bw/clustering"
	"github.com/james-lawrence/bw/cmd/bw/cmdopts"
	"github.com/james-lawrence/bw/uxterm"
	"google.golang.org/grpc"
)

type CmdControlQuorum struct {
	controlConnection
}

func (t CmdControlQuorum) Run(gctx *cmdopts.Global) (err error) {
	var (
		conn   *grpc.ClientConn
		d      dialers.Defaults
		c      clustering.Rendezvous
		quorum *agent.InfoResponse
	)

	if d, c, err = t.connect(gctx.Context); err != nil {
		return err
	}

	if conn, err = dialers.NewQuorum(c, d.Defaults()...).DialContext(gctx.Context); err != nil {
		return err
	}

	if quorum, err = agent.NewQuorumClient(conn).Info(gctx.Context, &agent.InfoRequest{}); err != nil {
		return err
	}

	if err = uxterm.PrintQuorum(quorum); err != nil {
		return err
	}

	return nil
}
