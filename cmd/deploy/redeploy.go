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
	"github.com/james-lawrence/bw/notary"
	"github.com/james-lawrence/bw/ux"
	"github.com/james-lawrence/bw/vcsinfo"
	"github.com/manifoldco/promptui"
	"github.com/pkg/errors"
	"google.golang.org/grpc"
)

func Redeploy(gctx *Context, deploymentID string) error {
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
	if config, err = commandutils.LoadConfiguration(gctx.Context, gctx.Environment, agent.CCOptionInsecure(gctx.Insecure)); err != nil {
		return err
	}

	displayname := vcsinfo.CurrentUserDisplay(config.WorkDir())

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

	if ss, err = notary.NewAutoSigner(displayname); err != nil {
		return err
	}

	events := make(chan *agent.Message, 100)
	local := commandutils.NewClientPeer(
		agent.PeerOptionName("local"),
	)

	var debugopt1 grpc.DialOption = grpc.EmptyDialOption{}
	var debugopt2 grpc.DialOption = grpc.EmptyDialOption{}

	if envx.Boolean(gctx.Debug, bw.EnvLogsGRPC, bw.EnvLogsVerbose) {
		debugopt1 = grpc.WithUnaryInterceptor(grpcx.DebugClientIntercepter)
		debugopt2 = grpc.WithStreamInterceptor(grpcx.DebugClientStreamIntercepter)
	}

	events <- agent.LogEvent(local, "connecting to cluster")
	if d, c, err = daemons.ConnectClientUntilSuccess(gctx.Context, config, ss, debugopt1, debugopt2, grpc.WithPerRPCCredentials(ss)); err != nil {
		return err
	}

	qd := dialers.NewQuorum(c, d.Defaults()...)

	if conn, err = qd.DialContext(gctx.Context); err != nil {
		return err
	}

	go func() {
		<-gctx.Context.Done()
		errorsx.Log(errors.Wrap(conn.Close(), "failed to close connection"))
	}()

	client = agent.NewDeployConn(conn)

	termui.NewFromClientConfig(
		gctx.Context, config, qd, local, events,
		ux.OptionHeartbeat(gctx.Heartbeat),
		ux.OptionDebug(gctx.Verbose),
	)

	events <- agent.LogEvent(local, "connected to cluster")
	go func() {
		<-gctx.Context.Done()
		if err = client.Close(); err != nil {
			log.Println("failed to close client", err)
		}
	}()

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

	events <- agent.LogEvent(local, fmt.Sprintf("located: who(%s) location(%s)", displayname, archive.Location))

	max := int64(config.Partitioner().Partition(len(cx.Members())))

	// only consider the canary node.
	if gctx.Canary {
		peers = agent.NodesToPeers(cx.Get(rendezvous.Auto()))
	} else {
		peers = cx.Peers()
	}

	peers = deployment.ApplyFilter(gctx.Filter, peers...)
	dopts := agent.DeployOptions{
		Concurrency:       max,
		Timeout:           int64(config.Deployment.Timeout),
		Heartbeat:         int64(gctx.Heartbeat),
		IgnoreFailures:    gctx.Lenient,
		SilenceDeployLogs: gctx.Silent,
	}

	if len(peers) == 0 && !gctx.AllowEmpty {
		cause := errorsx.String("deployment failed, filter did not match any servers")
		events <- agent.LogError(local, cause)
		return cause
	}

	events <- agent.LogEvent(local, fmt.Sprintf("initiating deploy: concurrency(%d), deployID(%s)", max, bw.RandomID(archive.DeploymentID)))
	if cause := client.RemoteDeploy(gctx.Context, displayname, &dopts, archive, peers...); cause != nil {
		events <- agent.LogEvent(local, fmt.Sprintln("deployment failed", cause))
	}

	return err
}
