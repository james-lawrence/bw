package agentcmd

import (
	"github.com/james-lawrence/bw/agent"
	"github.com/james-lawrence/bw/agent/dialers"
	"github.com/james-lawrence/bw/cmd/bw/cmdopts"
	"github.com/james-lawrence/bw/uxterm"
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

	conn, err := dialers.NewQuorum(c, d.Defaults()...).DialContext(gctx.Context)
	if err != nil {
		return err
	}
	defer conn.Close()

	info, err := agent.NewQuorumClient(conn).Info(gctx.Context, &agent.InfoRequest{})
	if err != nil {
		return err
	}

	return uxterm.PrintNodes(info, agent.NodesToPeers(t.filtered(c).Members()...))
}
