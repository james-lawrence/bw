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
		c       clustering.Rendezvous
		located *agent.Deploy
		archive *agent.Archive
		peers   []*agent.Peer
		ss      notary.Signer
	)

	log.Println("pid", os.Getpid())
	if config, err = commandutils.LoadConfiguration(ctx.Environment, agent.CCOptionInsecure(ctx.Insecure)); err != nil {
		return err
	}

	if len(config.Deployment.Prompt) > 0 {
		_, err := (&promptui.Prompt{
			Label:     config.Deployment.Prompt,
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
	local := commandutils.NewClientPeer(
		agent.PeerOptionName("local"),
	)

	var debugopt1 grpc.DialOption = grpc.EmptyDialOption{}
	var debugopt2 grpc.DialOption = grpc.EmptyDialOption{}

	if envx.Boolean(ctx.Debug, bw.EnvLogsGRPC, bw.EnvLogsVerbose) {
		debugopt1 = grpc.WithUnaryInterceptor(grpcx.DebugClientIntercepter)
		debugopt2 = grpc.WithStreamInterceptor(grpcx.DebugClientStreamIntercepter)
	}

	events <- agent.LogEvent(local, "connecting to cluster")
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

	termui.NewFromClientConfig(ctx.Context, config, d, local, events)

	events <- agent.LogEvent(local, "connected to cluster")
	go func() {
		<-ctx.Context.Done()
		if err = client.Close(); err != nil {
			log.Println("failed to close client", err)
		}
	}()

	go agentutil.WatchClusterEvents(ctx.Context, qd, local, events)

	cx := cluster.New(local, c)
	if located, err = agentutil.LocateDeployment(cx, qd, agentutil.FilterDeployID(deploymentID)); err != nil {
		events <- agent.LogError(local, errors.Wrap(err, "archive retrieval failed"))
		events <- agent.LogEvent(local, "deployment failed")
		return err
	}

	if located.Archive == nil {
		err = errors.New("archive retrieval failed, deployment found but archive is nil")
		events <- agent.LogError(local, err)
		events <- agent.LogEvent(local, "deployment failed")
		return err
	}

	archive = located.Archive

	events <- agent.LogEvent(local, fmt.Sprintf("located: who(%s) location(%s)", bw.DisplayName(), archive.Location))

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
		Timeout:           int64(config.Deployment.Timeout),
		IgnoreFailures:    ctx.Lenient,
		SilenceDeployLogs: ctx.Silent,
	}

	if len(peers) == 0 && !ctx.AllowEmpty {
		cause := errorsx.String("deployment failed, filter did not match any servers")
		events <- agent.LogError(local, cause)
		return cause
	}

	events <- agent.LogEvent(local, fmt.Sprintf("initiating deploy: concurrency(%d), deployID(%s)", max, bw.RandomID(archive.DeploymentID)))
	if cause := client.RemoteDeploy(ctx.Context, bw.DisplayName(), &dopts, archive, peers...); cause != nil {
		events <- agent.LogEvent(local, fmt.Sprintln("deployment failed", cause))
	}

	return err
}
