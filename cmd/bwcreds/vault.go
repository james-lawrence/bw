package main

import (
	"log"
	"os"

	"github.com/alecthomas/kingpin"
	"github.com/james-lawrence/bw"
	cc "github.com/james-lawrence/bw/certificatecache"
	"github.com/james-lawrence/bw/cmd/commandutils"
	"github.com/pkg/errors"
)

type vaultCreds struct {
	environment      string
	path             string
	commonName       string
	agentCredentials bool
}

func (t *vaultCreds) configure(parent *kingpin.CmdClause) {
	commandutils.EnvironmentArg(parent).StringVar(&t.environment)
	parent.Arg("path", "path to vault private key interface").StringVar(&t.path)
	parent.Arg("common-name", "common name for certificate, usually a domain name").StringVar(&t.commonName)
	parent.Flag("agent", "generate credentials for an agent, mainly used on servers").Default("false").BoolVar(&t.agentCredentials)
	parent.Action(t.generate)
}

func (t *vaultCreds) generate(ctx *kingpin.ParseContext) (err error) {
	path := bw.DefaultLocation(t.environment, "")

	if os.Geteuid() > 0 {
		path = bw.DefaultUserDirLocation(t.environment, "")
		log.Println("creating workspace configuration directory:", path)
		if err = os.MkdirAll(path, 0700); err != nil {
			return errors.WithStack(err)
		}
	}

	vcreds := cc.Vault{
		DefaultTokenFile: cc.VaultDefaultTokenPath(),
		CertificateDir:   path,
		CommonName:       t.commonName,
		Path:             t.path,
	}

	return cc.RefreshNow(path, vcreds)
}
