package agentcmd

import (
	"crypto/tls"
	"log"

	"github.com/davecgh/go-spew/spew"
	"github.com/james-lawrence/bw"
	"github.com/james-lawrence/bw/agent"
	"github.com/james-lawrence/bw/agent/dialers"
	"github.com/james-lawrence/bw/agent/discovery"
	"github.com/james-lawrence/bw/cmd/bw/cmdopts"
	"github.com/james-lawrence/bw/internal/errorsx"
	"github.com/james-lawrence/bw/internal/grpcx"
	"github.com/james-lawrence/bw/internal/tlsx"
	"github.com/james-lawrence/bw/muxer"
	"github.com/james-lawrence/bw/notary"
	"github.com/james-lawrence/bw/vcsinfo"
	"github.com/pkg/errors"
)

type CmdControlCheck struct {
	Insecure bool   `help:"skip tls verification"`
	Address  string `help:"address to check" arg:""`
}

func (t CmdControlCheck) Run(ctx *cmdopts.Global) (err error) {
	var (
		dd        dialers.Defaults
		ss        notary.Signer
		tlsconfig *tls.Config
	)

	if ss, err = notary.NewAutoSigner(vcsinfo.CurrentUserDisplay(".")); err != nil {
		return errors.Wrap(err, "unable to setup authorization")
	}

	tlsconfig, err = tlsx.Clone(&tls.Config{
		NextProtos:         []string{"bw.mux"},
		InsecureSkipVerify: t.Insecure,
	})
	if err != nil {
		return err
	}

	var di = discovery.ProxyDialer{
		Proxy:  t.Address,
		Signer: ss,
		Dialer: muxer.NewDialer(
			bw.ProtocolProxy,
			tlsx.NewDialer(tlsconfig),
		),
	}

	if dd, err = dialers.EndpointDialer(
		t.Address,
		di,
	); err != nil {
		return err
	}

	cc, err := dialers.NewDirect(agent.URIDiscovery(t.Address), dd.Defaults()...).DialContext(ctx.Context)
	if err != nil {
		return err
	}
	defer cc.Close()

	dc := discovery.NewDiscoveryClient(cc)

	resp, err := dc.Quorum(ctx.Context, &discovery.QuorumRequest{})
	if grpcx.IsUnavailable(err) {
		return errorsx.Notification(
			errors.Errorf("unable to connect to %s\nfor x509: certificate signed by unknown authority errors use --insecure to bypass\n\n%v\n", t.Address, err),
		)
	}

	if err != nil {
		return err
	}

	log.Println("quorum")
	for _, n := range resp.Nodes {
		log.Print(spew.Sdump(n))
	}

	return nil
}
