package main

import (
	"context"
	"eg/compute/debian"
	"eg/compute/maintainer"
	"log"

	"github.com/egdaemon/eg/runtime/wasi/eg"
	"github.com/egdaemon/eg/runtime/wasi/egenv"
	"github.com/egdaemon/eg/runtime/wasi/eggit"
)

func main() {
	ctx, done := context.WithTimeout(context.Background(), egenv.TTL())
	defer done()

	deb := eg.Container(maintainer.Container)
	err := eg.Perform(
		ctx,
		eggit.AutoClone,
		eg.Build(deb.BuildFromFile(".eg/Containerfile")),
		debian.Prepare,
		eg.Module(
			ctx,
			deb,
			debian.Build,
			debian.Upload,
		),
	)

	if err != nil {
		log.Fatalln(err)
	}
}
