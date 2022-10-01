package main

import (
	"log"

	"github.com/davecgh/go-spew/spew"
	"github.com/james-lawrence/bw"
	"github.com/james-lawrence/bw/agent"
	"github.com/james-lawrence/bw/agent/dialers"
	"github.com/james-lawrence/bw/clustering"
	"github.com/james-lawrence/bw/cmd/bw/cmdopts"
	"github.com/james-lawrence/bw/cmd/commandutils"
	"github.com/james-lawrence/bw/daemons"
	"github.com/james-lawrence/bw/notary"
	"google.golang.org/grpc"
)

// used to inspect permissions
type cmdNotary struct {
	Search cmdNotarySearch `cmd:"" help:"search users"`
}

type cmdNotarySearch struct {
	cmdopts.BeardedWookieEnv
	Insecure bool `help:"skip tls verification"`
}

func (t cmdNotarySearch) Run(ctx *cmdopts.Global) (err error) {
	var (
		d      dialers.Direct
		config agent.ConfigClient
		ss     notary.Signer
		c      clustering.Rendezvous
		s      notary.Notary_SearchClient
		page   *notary.SearchResponse
	)
	defer ctx.Shutdown()

	if config, err = commandutils.LoadConfiguration(t.Environment, agent.CCOptionInsecure(t.Insecure)); err != nil {
		return err
	}

	if ss, err = notary.NewAutoSigner(bw.DisplayName()); err != nil {
		return err
	}

	if d, c, err = daemons.Connect(config, ss, grpc.WithPerRPCCredentials(ss)); err != nil {
		return err
	}

	client := notary.NewClient(dialers.NewQuorum(c, d.Defaults()...))

	if s, err = client.Search(ctx.Context, &notary.SearchRequest{}); err != nil {
		return err
	}

	for page, err = s.Recv(); err == nil; page, err = s.Recv() {
		for _, g := range page.Grants {
			log.Println(g.Fingerprint, spew.Sdump(g.Permission))
		}
	}

	return err
}
