package agentcmd

import (
	"os"

	"github.com/james-lawrence/bw/agent"
	"github.com/james-lawrence/bw/agent/dialers"
	"github.com/james-lawrence/bw/cmd/bw/cmdopts"
	"github.com/james-lawrence/bw/cmd/commandutils"
	"github.com/james-lawrence/bw/internal/errorsx"
	"github.com/james-lawrence/bw/internal/jsonl"
	"github.com/james-lawrence/bw/ux"
	"github.com/pkg/errors"
)

// CmdControlHistory replays the messages recorded by the quorum.
type CmdControlHistory struct {
	controlConnection
	JSONL bool `name:"format-jsonl" help:"write each message as a json line to stdout instead of the styled output"`
}

func (t CmdControlHistory) Run(gctx *cmdopts.Global) (err error) {
	d, c, err := t.connect(gctx.Context)
	if err != nil {
		return err
	}

	conn, err := dialers.NewQuorum(c, d.Defaults()...).DialContext(gctx.Context)
	if err != nil {
		return err
	}
	defer func() { errorsx.Log(errors.Wrap(conn.Close(), "failed to close connection")) }()

	history, err := agent.NewQuorumClient(conn).History(gctx.Context, &agent.HistoryRequest{})
	if err != nil {
		return err
	}

	if t.JSONL {
		for _, m := range history.Messages {
			if err = jsonl.NewEncoder(os.Stdout).Encode(m); err != nil {
				return err
			}
		}
		return nil
	}

	ux.Replay(commandutils.NewClientPeer(), history.Messages)
	return nil
}
