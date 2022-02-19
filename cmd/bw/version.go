package main

import (
	"fmt"
	"runtime/debug"

	"github.com/james-lawrence/bw/cmd/bw/cmdopts"
)

type cmdVersion struct{}

func (t cmdVersion) Run(ctx *cmdopts.Global) (err error) {
	var (
		ok   bool
		info *debug.BuildInfo
	)

	if info, ok = debug.ReadBuildInfo(); ok {
		_, err = fmt.Println(info.Main.Path, info.Main.Version)
		return err
	}

	fmt.Println("unknown version")
	return nil
}
