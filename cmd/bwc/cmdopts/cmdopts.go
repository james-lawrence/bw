package cmdopts

import (
	"context"
	"sync"

	"github.com/james-lawrence/bw/cmd/commandutils"
)

type Global struct {
	Verbosity int                `help:"increase verbosity of logging" short:"v" type:"counter"`
	Context   context.Context    `kong:"-"`
	Shutdown  context.CancelFunc `kong:"-"`
	Cleanup   *sync.WaitGroup    `kong:"-"`
}

func (t Global) BeforeApply() error {
	commandutils.LogEnv(t.Verbosity)
	return nil
}
