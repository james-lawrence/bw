package main

import (
	"log"
	"os"

	"github.com/alecthomas/kingpin"
	"github.com/james-lawrence/bw/commands"
	"github.com/james-lawrence/bw/commands/commandutils"
)

// TODO:
// bwcreds awskms default {key-name}

func main() {
	var (
		err       error
		verbosity string
		ss        = &selfSigned{}
		vault     = &vaultCreds{}
	)

	app := kingpin.New("bearded-wookie", "deployment system").Version(commands.Version)
	app.Flag("verbosity", "verbosity level of errors").Short('v').Default(commandutils.VerbosityQuiet).
		Action(func(ctx *kingpin.ParseContext) (err error) {
			log.Println("configuring logs")
			commandutils.ConfigLog(verbosity)
			return nil
		}).
		EnumVar(&verbosity, commandutils.VerbosityQuiet, commandutils.VerbosityStack)
	ss.configure(app.Command("self-signed", "generate tls cert/key for an environment"))
	vault.configure(app.Command("vault", "generate tls cert/key for an environment using vault"))

	if _, err = app.Parse(os.Args[1:]); err != nil {
		commandutils.Fatalln(verbosity, err)
	}
}
