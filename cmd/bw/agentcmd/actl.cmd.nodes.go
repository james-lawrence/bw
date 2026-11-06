package agentcmd

import (
	"github.com/james-lawrence/bw/cmd/bw/cmdopts"
	"github.com/pterm/pterm"
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

	rendered, err := t.nodes(gctx.Context, d, c)
	if err != nil {
		return err
	}

	pterm.Println(rendered)
	return nil
}
