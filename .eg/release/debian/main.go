package main

import (
	"context"
	"eg/compute/maintainer"
	"log"

	"github.com/egdaemon/eg/runtime/wasi/eg"
	"github.com/egdaemon/eg/runtime/wasi/egdebuild"
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
		eg.Build(deb.BuildFromFile(".dist/deb/Dockerfile")),
		egdebuild.Prepare,
		eg.Module(
			ctx,
			deb,
			egdebuild.Build,
			egdebuild.Upload,
		),
	)

	if err != nil {
		log.Fatalln(err)
	}
}
