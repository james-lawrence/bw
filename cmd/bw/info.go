package main

import (
	"context"
	"io"
	"log"
	"os"
	"time"

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

	t.infoCmd(common(parent.Command("all", "retrieve info from all nodes within the cluster").Default()))
	t.logCmd(common(parent.Command("logs", "log retrieval for the latest deployment")))
	t.checkCmd(parent.Command("check", "check connectivity with the discovery service"))
}

func (t *agentInfo) infoCmd(parent *kingpin.CmdClause) *kingpin.CmdClause {
	return parent.Action(t.info)
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
		c      clustering.Cluster
		d      dialers.Defaults
		config agent.ConfigClient
		latest agent.Deploy
	)
	defer t.global.shutdown()

	if config, err = commandutils.LoadConfiguration(t.environment); err != nil {
		return err
	}

	log.Println("configuration", spew.Sdump(config))
	local := cluster.NewLocal(
		commandutils.NewClientPeer(),
		cluster.LocalOptionCapability(cluster.NewBitField(cluster.Passive)),
	)

	coptions := []daemons.ConnectOption{
		daemons.ConnectOptionClustering(
			clustering.OptionDelegate(local),
			clustering.OptionNodeID(local.Peer.Name),
			clustering.OptionBindAddress(local.Peer.Ip),
			clustering.OptionEventDelegate(cluster.LoggingEventHandler{}),
			clustering.OptionAliveDelegate(cluster.AliveDefault{}),
		),
	}

	if d, c, err = daemons.Connect(config, coptions...); err != nil {
		return err
	}

	cx := cluster.New(local, c)
	if latest, err = agentutil.DetermineLatestDeployment(cx, agent.NewDialer(d.Defaults()...)); err != nil {
		return err
	}

	logs := agentutil.DeploymentLogs(cx, agent.NewDialer(d.Defaults()...), latest.Archive.DeploymentID)
	return iox.Error(io.Copy(os.Stderr, logs))
}

func (t *agentInfo) info(ctx *kingpin.ParseContext) error {
	return t._info()
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
		log.Print(spew.Sdump(*n))
	}

	return nil
}

func (t *agentInfo) _info() (err error) {
	var (
		c      clustering.Cluster
		d      dialers.Defaults
		config agent.ConfigClient
		client agent.DeployClient
		ss     notary.Signer
	)
	defer t.global.shutdown()

	if config, err = commandutils.LoadConfiguration(t.environment); err != nil {
		return err
	}

	if ss, err = notary.NewAutoSigner(bw.DisplayName()); err != nil {
		return err
	}

	log.Println("configuration", spew.Sdump(config))
	local := cluster.NewLocal(
		commandutils.NewClientPeer(),
		cluster.LocalOptionCapability(cluster.NewBitField(cluster.Passive)),
	)

	coptions := []daemons.ConnectOption{
		daemons.ConnectOptionClustering(
			clustering.OptionDelegate(local),
			clustering.OptionNodeID(local.Peer.Name),
			clustering.OptionBindAddress(local.Peer.Ip),
			clustering.OptionEventDelegate(cluster.LoggingEventHandler{}),
			clustering.OptionAliveDelegate(cluster.AliveDefault{}),
		),
	}

	if d, c, err = daemons.Connect(config, coptions...); err != nil {
		return err
	}

	if client, err = agentutil.DeprecatedNewDeploy(config.Discovery, dialers.NewQuorum(c, d.Defaults(grpc.WithPerRPCCredentials(ss))...)); err != nil {
		return err
	}

	cx := cluster.New(local, c)
	err = agentutil.NewClusterOperation(agentutil.Operation(func(c agent.Client) (err error) {
		var (
			info agent.StatusResponse
		)

		if info, err = c.Info(); err != nil {
			return errors.WithStack(err)
		}

		log.Printf("Server: %s:%s - %s\n", info.Peer.Name, info.Peer.Ip, info.Peer.Status)
		log.Println("Previous Deployment: Time                          - DeploymentID               - Stage")
		for _, d := range info.Deployments {
			log.Printf("Previous Deployment: %s - %s - %s", time.Unix(d.Archive.Ts, 0).UTC(), bw.RandomID(d.Archive.DeploymentID), d.Stage)
		}

		return nil
	}))(cx, agent.NewDialer(d.Defaults()...))

	logx.MaybeLog(err)

	events := make(chan agent.Message, 100)

	t.global.cleanup.Add(1)
	go ux.Logging(t.global.ctx, t.global.cleanup, events, ux.OptionFailureDisplay(ux.NewFailureDisplayPrint(client)))
	log.Println("awaiting events")
	agentutil.WatchClusterEvents(t.global.ctx, config.Discovery, dialers.NewQuorum(cx, d.Defaults()...), local.Peer, events)

	return nil
}
