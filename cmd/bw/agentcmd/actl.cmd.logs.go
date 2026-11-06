package agentcmd

import (
	"io"
	"os"

	"github.com/james-lawrence/bw/agent"
	"github.com/james-lawrence/bw/agentutil"
	"github.com/james-lawrence/bw/cluster"
	"github.com/james-lawrence/bw/cmd/bw/cmdopts"
	"github.com/james-lawrence/bw/cmd/commandutils"
	"github.com/james-lawrence/bw/internal/iox"
)

type CmdControlLogs struct {
	controlConnection
}

func (t CmdControlLogs) Run(gctx *cmdopts.Global) (err error) {
	var (
		latest *agent.Deploy
	)
	defer gctx.Shutdown()

	d, c, err := t.connect(gctx.Context)
	if err != nil {
		return err
	}

	cx := cluster.New(commandutils.NewClientPeer(), c)
	if latest, err = agentutil.DetermineLatestDeployment(cx, d); err != nil {
		return err
	}

	logs := agentutil.DeploymentLogs(cx, d, latest.Archive.DeploymentID)
	return iox.Error(io.Copy(os.Stderr, logs))
}
