package main

import (
	"context"
	"io"
	"log"
	"os"

	"github.com/alecthomas/kingpin"
	"github.com/davecgh/go-spew/spew"
	"github.com/james-lawrence/bw"
	"github.com/james-lawrence/bw/agent"
	"github.com/james-lawrence/bw/agent/dialers"
	"github.com/james-lawrence/bw/agent/discovery"
	"github.com/james-lawrence/bw/agentutil"
	"github.com/james-lawrence/bw/cluster"
	"github.com/james-lawrence/bw/clustering"
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

type agentInfo struct {
	global       *global
	environment  string
	checkAddress string
}

func (t *agentInfo) configure(parent *kingpin.CmdClause) {
	common := func(cmd *kingpin.CmdClause) *kingpin.CmdClause {
		cmd.Arg("environment", "the environment configuration to use").Default(bw.DefaultEnvironmentName).StringVar(&t.environment)
		return cmd
	}

	t.watchCmd(common(parent.Command("watch", "watch cluster activity").Default()))
	t.nodesCmd(common(parent.Command("nodes", "retrieve info from ")))
	t.logCmd(common(parent.Command("logs", "log retrieval for the latest deployment")))
	t.checkCmd(parent.Command("check", "check connectivity with the discovery service"))
}

func (t *agentInfo) watchCmd(parent *kingpin.CmdClause) *kingpin.CmdClause {
	return parent.Action(t.watch)
}

func (t *agentInfo) nodesCmd(parent *kingpin.CmdClause) *kingpin.CmdClause {
	return parent.Action(t.nodes)
}

func (t *agentInfo) logCmd(parent *kingpin.CmdClause) *kingpin.CmdClause {
	return parent.Action(t.logs)
}

func (t *agentInfo) checkCmd(parent *kingpin.CmdClause) *kingpin.CmdClause {
	parent.Arg("address", "address to check").Required().StringVar(&t.checkAddress)
	return parent.Action(t.check)
}

func (t *agentInfo) logs(ctx *kingpin.ParseContext) (err error) {
	var (
		c      clustering.C
		d      dialers.Defaults
		config agent.ConfigClient
		latest *agent.Deploy
		ss     notary.Signer
	)
	defer t.global.shutdown()

	if config, err = commandutils.LoadConfiguration(t.environment); err != nil {
		return err
	}

	if ss, err = notary.NewAutoSigner(bw.DisplayName()); err != nil {
		return err
	}

	local := cluster.NewLocal(
		commandutils.NewClientPeer(),
		cluster.LocalOptionCapability(cluster.NewBitField(cluster.Passive)),
	)

	if d, c, err = daemons.Connect(config, grpc.WithPerRPCCredentials(ss)); err != nil {
		return err
	}

	cx := cluster.New(local, c)
	if latest, err = agentutil.DetermineLatestDeployment(cx, d); err != nil {
		return err
	}

	logs := agentutil.DeploymentLogs(cx, d, latest.Archive.DeploymentID)
	return iox.Error(io.Copy(os.Stderr, logs))
}

func (t *agentInfo) watch(ctx *kingpin.ParseContext) error {
	return t._watch()
}

func (t *agentInfo) nodes(ctx *kingpin.ParseContext) error {
	return t._nodes()
}

func (t *agentInfo) check(ctx *kingpin.ParseContext) (err error) {
	proxy := grpcx.NewCachedClient()
	cc, err := proxy.Dial(t.checkAddress, grpc.WithTransportCredentials(grpcx.InsecureTLS()))
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

func (t *agentInfo) _watch() (err error) {
	var (
		conn   *grpc.ClientConn
		c      clustering.C
		d      dialers.Direct
		config agent.ConfigClient
		quorum *agent.InfoResponse
		ss     notary.Signer
	)
	defer t.global.shutdown()

	if config, err = commandutils.LoadConfiguration(t.environment); err != nil {
		return err
	}

	if ss, err = notary.NewAutoSigner(bw.DisplayName()); err != nil {
		return err
	}

	local := cluster.NewLocal(
		commandutils.NewClientPeer(),
		cluster.LocalOptionCapability(cluster.NewBitField(cluster.Passive)),
	)

	if d, c, err = daemons.Connect(config, grpc.WithPerRPCCredentials(ss)); err != nil {
		return err
	}

	qd := dialers.NewQuorum(c, d.Defaults()...)

	if conn, err = qd.DialContext(t.global.ctx); err != nil {
		return err
	}

	go func() {
		<-t.global.ctx.Done()
		logx.MaybeLog(errors.Wrap(conn.Close(), "failed to close connection"))
	}()

	if quorum, err = agent.NewQuorumClient(conn).Info(t.global.ctx, &agent.InfoRequest{}); err != nil {
		return err
	}

	if err = uxterm.PrintQuorum(quorum); err != nil {
		return err
	}

	logx.MaybeLog(err)

	events := make(chan *agent.Message, 100)

	t.global.cleanup.Add(1)
	go ux.Logging(t.global.ctx, t.global.cleanup, events, ux.OptionFailureDisplay(ux.NewFailureDisplayPrint(qd)))
	log.Println("awaiting events")
	agentutil.WatchClusterEvents(t.global.ctx, qd, local.Peer, events)

	return nil
}

func (t *agentInfo) _nodes() (err error) {
	var (
		conn   *grpc.ClientConn
		c      clustering.C
		d      dialers.Defaults
		config agent.ConfigClient
		ss     notary.Signer
	)
	defer t.global.shutdown()

	if config, err = commandutils.LoadConfiguration(t.environment); err != nil {
		return err
	}

	if ss, err = notary.NewAutoSigner(bw.DisplayName()); err != nil {
		return err
	}

	local := cluster.NewLocal(
		commandutils.NewClientPeer(),
		cluster.LocalOptionCapability(cluster.NewBitField(cluster.Passive)),
	)

	if d, c, err = daemons.Connect(config, grpc.WithPerRPCCredentials(ss)); err != nil {
		return err
	}

	qd := dialers.NewQuorum(c, d.Defaults(grpc.WithPerRPCCredentials(ss))...)

	if conn, err = qd.DialContext(t.global.ctx); err != nil {
		return err
	}

	go func() {
		<-t.global.ctx.Done()
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
