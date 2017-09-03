package main

import (
	"io/ioutil"
	"log"
	"os"

	"bitbucket.org/jatone/bearded-wookie/archive"
	"bitbucket.org/jatone/bearded-wookie/downloads"

	"github.com/alecthomas/kingpin"
)

func main() {
	var (
		root string
		dst  string
	)
	app := kingpin.New("spike", "spike command line for testing functionality")
	app.Flag("root", "root directory to archive").StringVar(&root)
	app.Flag("dst", "dst directory to write into").StringVar(&dst)
	dlreg := downloads.New()
	app.Action(func(ctx *kingpin.ParseContext) error {
		var (
			pipe *os.File
			err  error
		)
		if pipe, err = ioutil.TempFile("", "archive"); err != nil {
			return err
		}
		defer os.Remove(pipe.Name())

		if err = archive.Pack(pipe, root); err != nil {
			return err
		}

		if err = pipe.Close(); err != nil {
			return err
		}
		src := dlreg.New("file://" + pipe.Name()).Download()
		defer src.Close()
		return archive.Unpack(dst, src)
	})
	// _ = app.Command("agent", "agent server").Action(agentx).Default()
	// _ = app.Command("client", "client cli").Action(deploy)

	if _, err := app.Parse(os.Args[1:]); err != nil {
		log.Println("boom", err)
	}
}
