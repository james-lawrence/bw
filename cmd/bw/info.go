package main

import (
	"context"
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
	"github.com/james-lawrence/bw/internal/x/grpcx"
	"github.com/james-lawrence/bw/internal/x/iox"
	"github.com/james-lawrence/bw/internal/x/logx"
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
	Check cmdInfoCheck `cmd:"" help:"check connectivity with the discovery service" hidden:"true"`
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
	go ux.Logging(ctx.Context, ctx.Cleanup, events, ux.OptionFailureDisplay(ux.NewFailureDisplayPrint(qd)))
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
			info agent.StatusResponse
		)

		if info, err = c.Info(); err != nil {
			return errors.WithStack(err)
		}

		return uxterm.PrintNode(&info)
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
	Address string `help:"address to check"`
}

func (t cmdInfoCheck) Run(ctx *cmdopts.Global) (err error) {
	proxy := grpcx.NewCachedClient()
	cc, err := proxy.Dial(t.Address, grpc.WithTransportCredentials(grpcx.InsecureTLS()))
	if err != nil {
		return err
	}

	resp, err := discovery.NewDiscoveryClient(cc).Quorum(context.Background(), &discovery.QuorumRequest{})
	if err != nil {
		return err
	}

	log.Println("quorum")
	for _, n := range resp.Nodes {
		log.Print(spew.Sdump(n))
	}

	return nil
}
