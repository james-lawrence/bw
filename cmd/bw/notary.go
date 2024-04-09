package main

import (
	"log"

	"github.com/bits-and-blooms/bloom/v3"
	"github.com/davecgh/go-spew/spew"
	"github.com/james-lawrence/bw/agent"
	"github.com/james-lawrence/bw/agent/dialers"
	"github.com/james-lawrence/bw/clustering"
	"github.com/james-lawrence/bw/cmd/bw/cmdopts"
	"github.com/james-lawrence/bw/cmd/commandutils"
	"github.com/james-lawrence/bw/daemons"
	"github.com/james-lawrence/bw/notary"
	"github.com/james-lawrence/bw/vcsinfo"
	"google.golang.org/grpc"
)

// used to inspect permissions
type cmdNotary struct {
	Search cmdNotarySearch `cmd:"" help:"search users"`
	Print  cmdNotaryPrint  `cmd:"" help:"list the fingerprints and their permissions in a file"`
}

type cmdNotarySearch struct {
	cmdopts.BeardedWookieEnv
	Insecure bool `help:"skip tls verification"`
}

func (t cmdNotarySearch) Run(gctx *cmdopts.Global) (err error) {
	var (
		d      dialers.Direct
		config agent.ConfigClient
		ss     notary.Signer
		c      clustering.Rendezvous
		s      notary.Notary_SearchClient
		page   *notary.SearchResponse
	)
	defer gctx.Shutdown()

	if config, err = commandutils.LoadConfiguration(gctx.Context, t.Environment, agent.CCOptionInsecure(t.Insecure)); err != nil {
		return err
	}

	displayname := vcsinfo.CurrentUserDisplay(config.WorkDir())

	if ss, err = notary.NewAutoSigner(displayname); err != nil {
		return err
	}

	if d, c, err = daemons.Connect(gctx.Context, config, ss, grpc.WithPerRPCCredentials(ss)); err != nil {
		return err
	}

	client := notary.NewClient(dialers.NewQuorum(c, d.Defaults()...))

	if s, err = client.Search(gctx.Context, &notary.SearchRequest{}); err != nil {
		return err
	}

	for page, err = s.Recv(); err == nil; page, err = s.Recv() {
		for _, g := range page.Grants {
			log.Println(g.Fingerprint, spew.Sdump(g.Permission))
		}
	}

	return err
}

type cmdNotaryPrint struct {
	Path string `help:"path of the file to inspect"`
}

func (t *cmdNotaryPrint) Run(ctx *cmdopts.Global) (err error) {
	var (
		n = notary.NewMem()
	)

	if err = notary.LoadAuthorizedKeys(n, t.Path); err != nil {
		return err
	}
	b := bloom.NewWithEstimates(1000, 0.0001)

	out := make(chan *notary.Grant)
	errc := make(chan error)
	go func() {
		select {
		case errc <- n.Sync(ctx.Context, b, out):
		case <-ctx.Context.Done():
			errc <- ctx.Context.Err()
		}
	}()

	for {
		select {
		case g := <-out:
			log.Println(g.Fingerprint, spew.Sdump(g.Permission))
		case err := <-errc:
			return err
		}
	}
}
