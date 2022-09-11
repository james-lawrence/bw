package deploy

import (
	"fmt"
	"log"
	"os"

	"github.com/james-lawrence/bw"
	"github.com/james-lawrence/bw/agent"
	"github.com/james-lawrence/bw/agent/dialers"
	"github.com/james-lawrence/bw/agentutil"
	"github.com/james-lawrence/bw/cluster"
	"github.com/james-lawrence/bw/clustering"
	"github.com/james-lawrence/bw/clustering/rendezvous"
	"github.com/james-lawrence/bw/cmd/commandutils"
	"github.com/james-lawrence/bw/cmd/termui"
	"github.com/james-lawrence/bw/daemons"
	"github.com/james-lawrence/bw/deployment"
	"github.com/james-lawrence/bw/internal/envx"
	"github.com/james-lawrence/bw/internal/errorsx"
	"github.com/james-lawrence/bw/internal/grpcx"
	"github.com/james-lawrence/bw/internal/logx"
	"github.com/james-lawrence/bw/notary"
	"github.com/manifoldco/promptui"
	"github.com/pkg/errors"
	"google.golang.org/grpc"
)

func Redeploy(ctx *Context, deploymentID string) error {
	var (
		err     error
		conn    *grpc.ClientConn
		d       dialers.Defaults
		client  agent.DeployClient
		config  agent.ConfigClient
		c       clustering.LocalRendezvous
		located *agent.Deploy
		archive *agent.Archive
		peers   []*agent.Peer
		ss      notary.Signer
	)

	log.Println("pid", os.Getpid())
	if config, err = commandutils.LoadConfiguration(ctx.Environment, agent.CCOptionInsecure(ctx.Insecure)); err != nil {
		return err
	}

	if len(config.DeployPrompt) > 0 {
		_, err := (&promptui.Prompt{
			Label:     config.DeployPrompt,
			IsConfirm: true,
		}).Run()

		// we're done.
		if err != nil {
			return nil
		}
	}

	if ss, err = notary.NewAutoSigner(bw.DisplayName()); err != nil {
		return err
	}

	events := make(chan *agent.Message, 100)
	local := cluster.NewLocal(
		commandutils.NewClientPeer(
			agent.PeerOptionName("local"),
		),
		cluster.LocalOptionCapability(cluster.NewBitField(cluster.Passive)),
	)

	var debugopt1 grpc.DialOption = grpc.EmptyDialOption{}
	var debugopt2 grpc.DialOption = grpc.EmptyDialOption{}

	if envx.Boolean(ctx.Debug, bw.EnvLogsGRPC, bw.EnvLogsVerbose) {
		debugopt1 = grpc.WithUnaryInterceptor(grpcx.DebugClientIntercepter)
		debugopt2 = grpc.WithStreamInterceptor(grpcx.DebugClientStreamIntercepter)
	}

	events <- agentutil.LogEvent(local.Peer, "connecting to cluster")
	if d, c, err = daemons.ConnectClientUntilSuccess(ctx.Context, config, ss, debugopt1, debugopt2, grpc.WithPerRPCCredentials(ss)); err != nil {
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

	client = agent.NewDeployConn(conn)

	termui.New(ctx.Context, ctx.CancelFunc, ctx.WaitGroup, qd, local.Peer, events)

	events <- agentutil.LogEvent(local.Peer, "connected to cluster")
	go func() {
		<-ctx.Context.Done()
		if err = client.Close(); err != nil {
			log.Println("failed to close client", err)
		}
	}()

	go agentutil.WatchClusterEvents(ctx.Context, qd, local.Peer, events)

	cx := cluster.New(local, c)
	if located, err = agentutil.LocateDeployment(cx, qd, agentutil.FilterDeployID(deploymentID)); err != nil {
		events <- agentutil.LogError(local.Peer, errors.Wrap(err, "archive retrieval failed"))
		events <- agentutil.LogEvent(local.Peer, "deployment failed")
		return err
	}

	if located.Archive == nil {
		err = errors.New("archive retrieval failed, deployment found but archive is nil")
		events <- agentutil.LogError(local.Peer, err)
		events <- agentutil.LogEvent(local.Peer, "deployment failed")
		return err
	}

	archive = located.Archive

	events <- agentutil.LogEvent(local.Peer, fmt.Sprintf("located: who(%s) location(%s)", archive.Initiator, archive.Location))

	max := int64(config.Partitioner().Partition(len(cx.Members())))

	// only consider the canary node.
	if ctx.Canary {
		peers = agent.NodesToPeers(cx.Get(rendezvous.Auto()))
	} else {
		peers = cx.Peers()
	}

	peers = deployment.ApplyFilter(ctx.Filter, peers...)
	dopts := agent.DeployOptions{
		Concurrency:       max,
		Timeout:           int64(config.DeployTimeout),
		IgnoreFailures:    ctx.Lenient,
		SilenceDeployLogs: ctx.Silent,
	}

	if len(peers) == 0 && !ctx.AllowEmpty {
		cause := errorsx.String("deployment failed, filter did not match any servers")
		events <- agentutil.LogError(local.Peer, cause)
		return cause
	}

	events <- agentutil.LogEvent(local.Peer, fmt.Sprintf("initiating deploy: concurrency(%d), deployID(%s)", max, bw.RandomID(archive.DeploymentID)))
	if cause := client.RemoteDeploy(ctx.Context, &dopts, archive, peers...); cause != nil {
		events <- agentutil.LogEvent(local.Peer, fmt.Sprintln("deployment failed", cause))
	}

	return err
}
