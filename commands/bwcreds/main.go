// Command bwcreds is used for setting up credentials from various
// sources, such as vault, aws kms, self-signed certificates.
package main

import (
	"log"
	"os"

	"github.com/alecthomas/kingpin"
	"github.com/james-lawrence/bw/commands"
	"github.com/james-lawrence/bw/commands/commandutils"
)

// bwcreds self-signed init
// bwcreds self-signed init talla.io *.talla.io 127.0.0.1 127.0.0.2
// bwcreds self-signed init talla.io wambli.talla.io
// bwcreds self-signed init talla.io wambli.talla.io
// bwcreds vault init pki/path
// TODO:
// bwcreds awskms init key-name

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
