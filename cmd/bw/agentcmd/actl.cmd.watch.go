package agentcmd

import (
	"context"
	"math"
	"time"

	"github.com/james-lawrence/bw/agent"
	"github.com/james-lawrence/bw/agent/dialers"
	"github.com/james-lawrence/bw/cmd/bw/cmdopts"
	"github.com/james-lawrence/bw/cmd/commandutils"
	"github.com/james-lawrence/bw/cmd/termui"
	"github.com/james-lawrence/bw/internal/errorsx"
	"github.com/james-lawrence/bw/ux"
	"github.com/james-lawrence/bw/uxterm"
	"github.com/pkg/errors"
)

type CmdControlWatch struct {
	controlConnection
}

func (t CmdControlWatch) Run(gctx *cmdopts.Global) (err error) {
	var (
		quorum *agent.InfoResponse
	)

	d, c, err := t.connect(gctx.Context)
	if err != nil {
		return err
	}

	local := commandutils.NewClientPeer()
	qd := dialers.NewQuorum(c, d.Defaults()...)

	conn, err := qd.DialContext(gctx.Context)
	if err != nil {
		return err
	}

	go func() {
		<-gctx.Context.Done()
		errorsx.Log(errors.Wrap(conn.Close(), "failed to close connection"))
	}()

	if quorum, err = agent.NewQuorumClient(conn).Info(gctx.Context, &agent.InfoRequest{}); err != nil {
		return err
	}

	if err = uxterm.PrintQuorum(quorum); err != nil {
		return err
	}

	events := make(chan *agent.Message, 100)

	dctx, failurefn := context.WithCancelCause(gctx.Context)
	defer failurefn(nil)
	termui.NewLogging(
		dctx,
		failurefn,
		qd,
		local,
		events,
		ux.OptionFailureDisplay(ux.NewFailureDisplayPrint(local, qd)),
		ux.OptionHeartbeat(time.Duration(math.MaxInt64)),
	)

	<-dctx.Done()

	if err = context.Cause(dctx); errorsx.Ignore(err, context.Canceled) != nil {
		return err
	}

	return nil
}
