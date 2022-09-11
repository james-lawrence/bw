package main

import (
	"crypto/tls"
	"io"
	"log"
	"os"

	"github.com/davecgh/go-spew/spew"
	"github.com/james-lawrence/bw"
	"github.com/james-lawrence/bw/agent"
	"github.com/james-lawrence/bw/agent/dialers"
	"github.com/james-lawrence/bw/agent/discovery"
	"github.com/james-lawrence/bw/agentutil"
	"github.com/james-lawrence/bw/cluster"
	"github.com/james-lawrence/bw/clustering"
	"github.com/james-lawrence/bw/cmd/bw/cmdopts"
	"github.com/james-lawrence/bw/cmd/commandutils"
	"github.com/james-lawrence/bw/daemons"
	"github.com/james-lawrence/bw/internal/errorsx"
	"github.com/james-lawrence/bw/internal/grpcx"
	"github.com/james-lawrence/bw/internal/iox"
	"github.com/james-lawrence/bw/internal/logx"
	"github.com/james-lawrence/bw/internal/tlsx"
	"github.com/james-lawrence/bw/muxer"
	"github.com/james-lawrence/bw/notary"
	"github.com/james-lawrence/bw/ux"
	"github.com/james-lawrence/bw/uxterm"

	"github.com/pkg/errors"
	"google.golang.org/grpc"
)

type cmdInfo struct {
	Watch cmdInfoWatch `cmd:"" help:"watch cluster activity"`
	Nodes cmdInfoNodes `cmd:"" help:"retrieve nodes within the cluster"`
	Logs  cmdInfoLogs  `cmd:"" help:"log retrieval for the latest deployment"`
	Check cmdInfoCheck `cmd:"" help:"check connectivity with the discovery service"`
}

type cmdInfoWatch struct {
	cmdopts.BeardedWookieEnv
	Insecure bool `help:"skip tls verification"`
}

func (t cmdInfoWatch) Run(ctx *cmdopts.Global) (err error) {
	var (
		conn   *grpc.ClientConn
		c      clustering.C
		d      dialers.Direct
		config agent.ConfigClient
		quorum *agent.InfoResponse
		ss     notary.Signer
	)
	defer ctx.Shutdown()

	if config, err = commandutils.LoadConfiguration(t.Environment, agent.CCOptionInsecure(t.Insecure)); err != nil {
		return err
	}

	if ss, err = notary.NewAutoSigner(bw.DisplayName()); err != nil {
		return err
	}

	local := cluster.NewLocal(
		commandutils.NewClientPeer(),
		cluster.LocalOptionCapability(cluster.NewBitField(cluster.Passive)),
	)

	if d, c, err = daemons.Connect(config, ss, grpc.WithPerRPCCredentials(ss)); err != nil {
		return err
	}

	qd := dialers.NewQuorum(c, d.Defaults()...)

	if conn, err = qd.DialContext(ctx.Context); err != nil {
		return err
	}

	go func() {
		<-ctx.Context.Done()
		logx.MaybeLog(errors.Wrap(conn.Close(), "failed to close connection"))
	}()

	if quorum, err = agent.NewQuorumClient(conn).Info(ctx.Context, &agent.InfoRequest{}); err != nil {
		return err
	}

	if err = uxterm.PrintQuorum(quorum); err != nil {
		return err
	}

	logx.MaybeLog(err)

	events := make(chan *agent.Message, 100)

	ctx.Cleanup.Add(1)
	go ux.Logging(ctx.Context, ctx.Cleanup, events, ux.OptionFailureDisplay(ux.NewFailureDisplayPrint(local.Peer, qd)))
	log.Println("awaiting events")
	agentutil.WatchClusterEvents(ctx.Context, qd, local.Peer, events)

	return nil
}

type cmdInfoNodes struct {
	cmdopts.BeardedWookieEnv
	Insecure bool `help:"skip tls verification"`
}

func (t cmdInfoNodes) Run(ctx *cmdopts.Global) (err error) {
	var (
		conn   *grpc.ClientConn
		c      clustering.C
		d      dialers.Defaults
		config agent.ConfigClient
		ss     notary.Signer
	)
	defer ctx.Shutdown()

	if config, err = commandutils.LoadConfiguration(t.Environment, agent.CCOptionInsecure(t.Insecure)); err != nil {
		return err
	}

	if ss, err = notary.NewAutoSigner(bw.DisplayName()); err != nil {
		return err
	}

	local := cluster.NewLocal(
		commandutils.NewClientPeer(),
		cluster.LocalOptionCapability(cluster.NewBitField(cluster.Passive)),
	)

	if d, c, err = daemons.Connect(config, ss, grpc.WithPerRPCCredentials(ss)); err != nil {
		return err
	}

	qd := dialers.NewQuorum(c, d.Defaults()...)

	if conn, err = qd.DialContext(ctx.Context); err != nil {
		return err
	}

	go func() {
		<-ctx.Context.Done()
		logx.MaybeLog(errors.Wrap(conn.Close(), "failed to close connection"))
	}()

	cx := cluster.New(local, c)
	return agentutil.NewClusterOperation(agentutil.Operation(func(c agent.Client) (err error) {
		var (
			info *agent.StatusResponse
		)

		if info, err = c.Info(ctx.Context); err != nil {
			return errors.WithStack(err)
		}

		return uxterm.PrintNode(info)
	}))(cx, d)
}

type cmdInfoLogs struct {
	cmdopts.BeardedWookieEnv
	Insecure bool `help:"skip tls verification"`
}

func (t cmdInfoLogs) Run(ctx *cmdopts.Global) (err error) {
	var (
		c      clustering.C
		d      dialers.Defaults
		config agent.ConfigClient
		latest *agent.Deploy
		ss     notary.Signer
	)
	defer ctx.Shutdown()

	if config, err = commandutils.LoadConfiguration(t.Environment, agent.CCOptionInsecure(t.Insecure)); err != nil {
		return err
	}

	if ss, err = notary.NewAutoSigner(bw.DisplayName()); err != nil {
		return err
	}

	local := cluster.NewLocal(
		commandutils.NewClientPeer(),
		cluster.LocalOptionCapability(cluster.NewBitField(cluster.Passive)),
	)

	if d, c, err = daemons.Connect(config, ss, grpc.WithPerRPCCredentials(ss)); err != nil {
		return err
	}

	cx := cluster.New(local, c)
	if latest, err = agentutil.DetermineLatestDeployment(cx, d); err != nil {
		return err
	}

	logs := agentutil.DeploymentLogs(cx, d, latest.Archive.DeploymentID)
	return iox.Error(io.Copy(os.Stderr, logs))
}

type cmdInfoCheck struct {
	Insecure bool   `help:"skip tls verification"`
	Address  string `help:"address to check" arg:""`
}

func (t cmdInfoCheck) Run(ctx *cmdopts.Global) (err error) {
	var (
		dd        dialers.Defaults
		ss        notary.Signer
		tlsconfig *tls.Config
	)

	if ss, err = notary.NewAutoSigner(bw.DisplayName()); err != nil {
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

	if dd, err = dialers.DefaultDialer(
		t.Address,
		di,
	); err != nil {
		return err
	}

	cc, err := dialers.NewDirect(agent.URIDiscovery(t.Address), dd.Defaults()...).DialContext(ctx.Context)
	if err != nil {
		return err
	}
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
