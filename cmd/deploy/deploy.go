package deploy

import (
	"context"
	"fmt"
	"io"
	"log"
	"os"
	"path/filepath"
	"sync"
	"time"

	"github.com/james-lawrence/bw"
	"github.com/james-lawrence/bw/agent"
	"github.com/james-lawrence/bw/agent/dialers"
	"github.com/james-lawrence/bw/archive"
	"github.com/james-lawrence/bw/clustering"
	"github.com/james-lawrence/bw/clustering/rendezvous"
	"github.com/james-lawrence/bw/cmd/commandutils"
	"github.com/james-lawrence/bw/cmd/termui"
	"github.com/james-lawrence/bw/daemons"
	"github.com/james-lawrence/bw/deployment"
	"github.com/james-lawrence/bw/internal/envx"
	"github.com/james-lawrence/bw/internal/errorsx"
	"github.com/james-lawrence/bw/internal/grpcx"
	"github.com/james-lawrence/bw/internal/iox"
	"github.com/james-lawrence/bw/notary"
	"github.com/james-lawrence/bw/ux"
	"github.com/james-lawrence/bw/vcsinfo"
	"github.com/manifoldco/promptui"
	"github.com/pkg/errors"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
)

type Context struct {
	Environment string
	Concurrency int64
	Filter      deployment.Filter
	Verbose     bool
	Insecure    bool
	Lenient     bool
	Silent      bool
	Heartbeat   time.Duration
	AllowEmpty  bool
	Canary      bool
	Debug       bool
	context.Context
	context.CancelFunc
	*sync.WaitGroup
}

// Into deploy into the specified environment.
func Into(gctx *Context) error {
	var (
		err       error
		dst       *os.File
		dstinfo   os.FileInfo
		conn      *grpc.ClientConn
		d         dialers.Defaults
		client    agent.DeployClient
		config    agent.ConfigClient
		c         clustering.Rendezvous
		ss        notary.Signer
		darchive  *agent.Archive
		peers     []*agent.Peer
		commitish string
	)

	if config, err = commandutils.LoadConfiguration(gctx.Context, gctx.Environment, agent.CCOptionInsecure(gctx.Insecure)); err != nil {
		return errors.Wrap(err, "unable to load configuration")
	}

	displayname := vcsinfo.CurrentUserDisplay(config.WorkDir())

	if ss, err = notary.NewAutoSigner(displayname); err != nil {
		return errors.Wrap(err, "unable to setup authorization")
	}

	log.Println("pid", os.Getpid())

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

	if commitish, err = commandutils.RunLocalDirectives(gctx.Context, config); err != nil {
		return errors.Wrap(err, "failed to run local directives")
	}

	if !commandutils.RemoteTasksAvailable(config) {
		log.Println("no directives to run by the cluster")
		return nil
	}

	events := make(chan *agent.Message, 100)

	local := commandutils.NewClientPeer(
		agent.PeerOptionName("local"),
	)

	events <- agent.LogEvent(local, "connecting to cluster")
	var (
		debugopt1 grpc.DialOption = grpc.EmptyDialOption{}
		debugopt2 grpc.DialOption = grpc.EmptyDialOption{}
	)

	if gctx.Debug || gctx.Verbose || envx.Boolean(false, bw.EnvLogsGRPC, bw.EnvLogsVerbose) {
		debugopt1 = grpc.WithUnaryInterceptor(grpcx.DebugClientIntercepter)
		debugopt2 = grpc.WithStreamInterceptor(grpcx.DebugClientStreamIntercepter)
	}

	if d, c, err = daemons.ConnectClientUntilSuccess(gctx.Context, config, ss, debugopt1, debugopt2, grpc.WithPerRPCCredentials(ss)); err != nil {
		return errors.Wrap(err, "unable to connect to cluster")
	}

	qd := dialers.NewQuorum(c, d.Defaults()...)

	termui.NewFromClientConfig(
		gctx.Context, config, qd, local, events,
		ux.OptionHeartbeat(gctx.Heartbeat),
		ux.OptionDebug(gctx.Verbose),
	)

	conn = grpcx.UntilSuccess(gctx.Context, func(ictx context.Context) (*grpc.ClientConn, error) {
		return qd.DialContext(ictx)
	})

	go func() {
		<-gctx.Context.Done()
		errorsx.Log(errors.Wrap(conn.Close(), "failed to close connection"))
	}()

	client = agent.NewDeployConn(conn)

	events <- agent.LogEvent(local, "connected to cluster")

	deployspace := config.Deployspace()
	if err = os.WriteFile(filepath.Join(deployspace, bw.EnvFile), []byte(config.Environment), 0600); err != nil {
		return err
	}

	if _, err := os.Stat(filepath.Join(config.Dir(), bw.AuthKeysFile)); !os.IsNotExist(err) {
		if err = iox.Copy(filepath.Join(config.Dir(), bw.AuthKeysFile), filepath.Join(deployspace, bw.AuthKeysFile)); err != nil {
			return err
		}
	}

	if dst, err = os.CreateTemp("", "bwarchive"); err != nil {
		events <- agent.LogError(local, errors.Wrap(err, "archive creation failed"))
		events <- agent.LogEvent(local, "deployment failed")
		return nil
	}
	defer os.Remove(dst.Name())
	defer dst.Close()

	if err = archive.Pack(dst, deployspace); err != nil {
		return err
	}

	if dstinfo, err = dst.Stat(); err != nil {
		events <- agent.LogError(local, errors.Wrap(err, "archive creation failed"))
		events <- agent.LogEvent(local, "deployment failed")
		return nil
	}

	events <- agent.LogEvent(local, "archive upload initiated")
	err = grpcx.Retry(gctx.Context, func() error {
		if _, err = dst.Seek(0, io.SeekStart); err != nil {
			events <- agent.LogError(local, errors.Wrap(err, "archive creation failed"))
			events <- agent.LogEvent(local, "deployment failed")
			return nil
		}

		meta := agent.UploadMetadata{
			Bytes:     uint64(dstinfo.Size()),
			Vcscommit: commitish,
		}

		if darchive, err = client.Upload(gctx.Context, &meta, dst); err != nil {
			events <- agent.LogError(local, errors.Wrap(err, "archive upload failed"))
			events <- agent.LogEvent(local, "deployment failed")
			return err
		}

		return nil
	}, codes.Unavailable)

	if err != nil {
		return err
	}

	events <- agent.LogEvent(local, fmt.Sprintf("archive upload completed: who(%s) location(%s)", displayname, darchive.Location))

	max := gctx.Concurrency
	if gctx.Concurrency == 0 {
		max = int64(config.Partitioner().Partition(len(c.Members())))
	}

	// only consider the canary node.
	if gctx.Canary {
		peers = agent.NodesToPeers(c.Get(rendezvous.Auto()))
	} else {
		peers = agent.NodesToPeers(c.Members()...)
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

	events <- agent.LogEvent(local, fmt.Sprintf("deploy initiated: by(%s) concurrency(%d), deployID(%s)", displayname, max, bw.RandomID(darchive.DeploymentID)))
	if cause := client.RemoteDeploy(gctx.Context, displayname, &dopts, darchive, peers...); cause != nil {
		events <- agent.LogError(local, errors.Wrap(cause, "deploy failed"))
		events <- agent.DeployEventFailed(local, displayname, &dopts, darchive, cause)
		events <- agent.NewDeployCommand(local, agent.DeployCommandFailed(displayname, darchive.DeployOption, dopts.DeployOption))
	}

	return err
}
