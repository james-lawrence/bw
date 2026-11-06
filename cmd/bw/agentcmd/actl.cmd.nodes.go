package agentcmd

import (
	"context"

	"github.com/james-lawrence/bw/agent"
	"github.com/james-lawrence/bw/agentutil"
	"github.com/james-lawrence/bw/cluster"
	"github.com/james-lawrence/bw/cmd/bw/cmdopts"
	"github.com/james-lawrence/bw/cmd/commandutils"
	"github.com/james-lawrence/bw/uxterm"
	"github.com/pkg/errors"
)

type CmdControlNodes struct {
	controlConnection
}

func (t CmdControlNodes) Run(gctx *cmdopts.Global) (err error) {
	defer gctx.Shutdown()

	d, c, err := t.connect(gctx.Context)
	if err != nil {
		return err
	}

	cx := cluster.New(commandutils.NewClientPeer(), t.filtered(c))
	return agentutil.NewClusterOperation(gctx.Context, agentutil.Operation(func(ctx context.Context, p *agent.Peer, c agent.Client) (err error) {
		var (
			info *agent.StatusResponse
		)

		if info, err = c.Info(ctx); err != nil {
			return errors.WithStack(err)
		}

		return uxterm.PrintNode(info)
	}))(cx, d)
}
