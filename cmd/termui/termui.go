package termui

import (
	"context"
	"sync"

	"github.com/james-lawrence/bw/agent"
	"github.com/james-lawrence/bw/agent/dialers"
	"github.com/james-lawrence/bw/ux"
)

func New(ctx context.Context, shutdown context.CancelFunc, wg *sync.WaitGroup, c dialers.Defaults, events chan *agent.Message) {
	wg.Add(1)
	go func() {
		ux.Deploy(ctx, wg, events, ux.OptionFailureDisplay(ux.NewFailureDisplayPrint(c)))
		shutdown()
	}()
}
