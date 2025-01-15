package main

import (
	"context"
	"eg/compute/daemon"
	"eg/compute/integration"
	"log"

	"github.com/egdaemon/eg/runtime/wasi/eg"
	"github.com/egdaemon/eg/runtime/wasi/egenv"
	"github.com/egdaemon/eg/runtime/wasi/eggit"
)

func main() {
	ctx, done := context.WithTimeout(context.Background(), egenv.TTL())
	defer done()

	err := eg.Perform(
		ctx,
		eggit.AutoClone,
		daemon.Install,
		eg.Parallel(
			daemon.Test,
			integration.Test,
		),
	)

	if err != nil {
		log.Fatalln(err)
	}
}
