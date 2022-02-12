package main

import (
	"os"

	"github.com/alecthomas/kingpin"
	"github.com/james-lawrence/bw/cmd"
	"github.com/james-lawrence/bw/cmd/commandutils"
)

func main() {
	var (
		err       error
		verbosity int
		ss        = &selfSigned{}
		vault     = &vaultCreds{}
	)

	app := kingpin.New("bearded-wookie", "deployment system").Version(cmd.Version)
	app.Flag("verbose", "increase verbosity of logging").Short('v').Default("0").Action(func(*kingpin.ParseContext) error {
		commandutils.LogEnv(verbosity)
		return nil
	}).CounterVar(&verbosity)
	ss.configure(app.Command("self-signed", "generate tls cert/key for an environment"))
	vault.configure(app.Command("vault", "generate tls cert/key for an environment using vault"))

	if _, err = app.Parse(os.Args[1:]); err != nil {
		commandutils.LogCause(err)
	}
}
