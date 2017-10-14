package main

import (
	"context"
	"log"
	"os"
	"sync"

	"bitbucket.org/jatone/bearded-wookie"
	"bitbucket.org/jatone/bearded-wookie/agent"
	"bitbucket.org/jatone/bearded-wookie/agentutil"
	"bitbucket.org/jatone/bearded-wookie/ux"
	"bitbucket.org/jatone/bearded-wookie/x/logx"
	"bitbucket.org/jatone/bearded-wookie/x/systemx"

	"github.com/alecthomas/kingpin"
)

func main() {
	background := &sync.WaitGroup{}
	ctx, done := context.WithCancel(context.Background())
	app := kingpin.New("spike", "spike command line for testing functionality")
	app.Command("example1", "example 1").Action(example1(ctx, done, background))
	if _, err := app.Parse(os.Args[1:]); err != nil {
		log.Fatalln(err)
	}

	systemx.Cleanup(ctx, done, background, os.Kill, os.Interrupt)(func() {
		log.Println("waiting for systems to shutdown")
	})
}

func example1(ctx context.Context, done context.CancelFunc, background *sync.WaitGroup) func(*kingpin.ParseContext) error {
	return func(*kingpin.ParseContext) (err error) {
		local := agent.LocalPeer(bw.MustGenerateID().String())
		events := make(chan agent.Message, 100)
		ux.NewTermui(ctx, done, background, events)
		d := agentutil.NewBusDispatcher(events)

		logx.MaybeLog(d.Dispatch(agentutil.PeersFoundEvent(local, 1)))
		return nil
	}
}
