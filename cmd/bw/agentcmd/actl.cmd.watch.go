package agentcmd

import (
	"context"
	"math"
	"time"

	"github.com/james-lawrence/bw/agent"
	"github.com/james-lawrence/bw/agent/dialers"
	"github.com/james-lawrence/bw/clustering"
	"github.com/james-lawrence/bw/cmd/bw/cmdopts"
	"github.com/james-lawrence/bw/cmd/commandutils"
	"github.com/james-lawrence/bw/cmd/termui"
	"github.com/james-lawrence/bw/internal/errorsx"
	"github.com/james-lawrence/bw/ux"
	"github.com/pkg/errors"
	"github.com/pterm/pterm"
)

type CmdControlWatch struct {
	controlConnection
	Refresh time.Duration `name:"refresh" help:"how often to check the nodes of the cluster, the nodes are printed again when they change. 0 only prints them once" default:"15s"`
}

func (t CmdControlWatch) Run(gctx *cmdopts.Global) (err error) {
	d, c, err := t.connect(gctx.Context)
	if err != nil {
		return err
	}

	local := commandutils.NewClientPeer()
	qd := dialers.NewQuorum(c, d.Defaults()...)

	rendered, err := t.nodes(gctx.Context, d, c)
	if err != nil {
		return err
	}
	pterm.Println(rendered)

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

	if t.Refresh > 0 {
		go t.refresh(dctx, d, c, rendered)
	}

	<-dctx.Done()

	if err = context.Cause(dctx); errorsx.Ignore(err, context.Canceled) != nil {
		return err
	}

	return nil
}

// refresh prints the nodes again whenever they differ from last, until the context is done.
func (t CmdControlWatch) refresh(ctx context.Context, d dialers.Defaults, c clustering.Rendezvous, last string) {
	for ticker := time.NewTicker(t.Refresh); ; {
		select {
		case <-ctx.Done():
			ticker.Stop()
			return
		case <-ticker.C:
			rendered, err := t.nodes(ctx, d, c)
			if err != nil {
				errorsx.Log(errors.Wrap(err, "unable to refresh the nodes"))
				continue
			}

			if rendered == last {
				continue
			}

			last = rendered
			pterm.Println()
			pterm.Println(rendered)
		}
	}
}
