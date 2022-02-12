// Package bwc is the user client which focuses on human friendly behaviors not system administration, and not on backwards compatibility.
package main

import (
	"io/fs"
	"os"
	"path/filepath"
	"strings"

	"github.com/alecthomas/kong"
	"github.com/james-lawrence/bw"
	"github.com/posener/complete"
	"github.com/willabides/kongplete"
)

type Global struct{}

var shellCli struct {
	Deploy             deployCmd                    `cmd:"" help:"deployment related commands"`
	InstallCompletions kongplete.InstallCompletions `cmd:"" help:"install shell completions"`
}

func main() {
	parser := kong.Must(
		&shellCli,
		kong.Name("bwc"),
		kong.Description("user frontend to bearded-wookie"),
		kong.UsageOnError(),
	)

	// Run kongplete.Complete to handle completion requests
	kongplete.Complete(parser,
		kongplete.WithPredictor("bw.environment", complete.PredictFunc(func(args complete.Args) (results []string) {
			root := bw.LocateDeployspace(bw.DefaultDeployspaceConfigDir)
			filepath.WalkDir(root, func(path string, d fs.DirEntry, err error) error {
				if root == path {
					return nil
				}

				if !d.IsDir() {
					return nil
				}

				name := d.Name()

				if strings.HasPrefix(name, args.Last) {
					results = append(results, name)
				}

				return filepath.SkipDir
			})
			return results
		})),
		kongplete.WithPredictor("file", complete.PredictFiles("*")),
	)

	ctx, err := parser.Parse(os.Args[1:])
	if err != nil {
		ctx.FatalIfErrorf(ctx.Error)
	}

	ctx.FatalIfErrorf(ctx.Run(&Global{}))
}
