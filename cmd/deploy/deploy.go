package deploy

import (
	"context"
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"os"
	"path/filepath"
	"sync"

	"github.com/james-lawrence/bw"
	"github.com/james-lawrence/bw/agent"
	"github.com/james-lawrence/bw/agent/dialers"
	"github.com/james-lawrence/bw/agentutil"
	"github.com/james-lawrence/bw/archive"
	"github.com/james-lawrence/bw/clustering"
	"github.com/james-lawrence/bw/clustering/rendezvous"
	"github.com/james-lawrence/bw/cmd/commandutils"
	"github.com/james-lawrence/bw/cmd/termui"
	"github.com/james-lawrence/bw/daemons"
	"github.com/james-lawrence/bw/deployment"
	"github.com/james-lawrence/bw/internal/x/errorsx"
	"github.com/james-lawrence/bw/internal/x/grpcx"
	"github.com/james-lawrence/bw/internal/x/iox"
	"github.com/james-lawrence/bw/internal/x/logx"
	"github.com/james-lawrence/bw/notary"
	"github.com/manifoldco/promptui"
	"github.com/pkg/errors"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
)

type Context struct {
	Environment    string
	Concurrency    int64
	Filter         deployment.Filter
	Insecure       bool
	IgnoreFailures bool
	SilenceLogs    bool
	AllowEmpty     bool
	Canary         bool
	context.Context
	context.CancelFunc
	*sync.WaitGroup
}

// Into deploy into the specified environment.
func Into(ctx *Context) error {
	var (
		err      error
		dst      *os.File
		dstinfo  os.FileInfo
		conn     *grpc.ClientConn
		d        dialers.Defaults
		client   agent.DeployClient
		config   agent.ConfigClient
		c        clustering.LocalRendezvous
		ss       notary.Signer
		darchive agent.Archive
		peers    []*agent.Peer
	)

	if ss, err = notary.NewAutoSigner(bw.DisplayName()); err != nil {
		return errors.Wrap(err, "unable to setup authorization")
	}

	if config, err = commandutils.LoadConfiguration(ctx.Environment, agent.CCOptionInsecure(ctx.Insecure)); err != nil {
		return errors.Wrap(err, "unable to load configuration")
	}

	log.Println("pid", os.Getpid())

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

	if err = commandutils.RunLocalDirectives(config); err != nil {
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

	events <- agentutil.LogEvent(local, "connecting to cluster")
	if d, c, err = daemons.ConnectClientUntilSuccess(ctx.Context, config, ss, grpc.WithPerRPCCredentials(ss)); err != nil {
		return errors.Wrap(err, "unable to connect to cluster")
	}

	qd := dialers.NewQuorum(c, d.Defaults()...)

	if conn, err = qd.DialContext(ctx.Context); err != nil {
		return errors.Wrap(err, "unable to create a connection")
	}

	go func() {
		<-ctx.Context.Done()
		logx.MaybeLog(errors.Wrap(conn.Close(), "failed to close connection"))
	}()

	client = agent.NewDeployConn(conn)

	termui.New(ctx.Context, ctx.CancelFunc, ctx.WaitGroup, d, events)
	events <- agentutil.LogEvent(local, "connected to cluster")

	go agentutil.WatchClusterEvents(ctx.Context, qd, local, events)

	if err = ioutil.WriteFile(filepath.Join(config.DeployDataDir, bw.EnvFile), []byte(config.Environment), 0600); err != nil {
		return err
	}

	if _, err := os.Stat(filepath.Join(config.Dir(), bw.AuthKeysFile)); !os.IsNotExist(err) {
		if err = iox.Copy(filepath.Join(config.Dir(), bw.AuthKeysFile), filepath.Join(config.DeployDataDir, bw.AuthKeysFile)); err != nil {
			return err
		}
	}

	if dst, err = ioutil.TempFile("", "bwarchive"); err != nil {
		events <- agentutil.LogError(local, errors.Wrap(err, "archive creation failed"))
		events <- agentutil.LogEvent(local, "deployment failed")
		return nil
	}
	defer os.Remove(dst.Name())
	defer dst.Close()

	if err = archive.Pack(dst, config.DeployDataDir); err != nil {
		return err
	}

	if dstinfo, err = dst.Stat(); err != nil {
		events <- agentutil.LogError(local, errors.Wrap(err, "archive creation failed"))
		events <- agentutil.LogEvent(local, "deployment failed")
		return nil
	}

	events <- agentutil.LogEvent(local, "archive upload initiated")
	err = grpcx.Retry(func() error {
		if _, err = dst.Seek(0, io.SeekStart); err != nil {
			events <- agentutil.LogError(local, errors.Wrap(err, "archive creation failed"))
			events <- agentutil.LogEvent(local, "deployment failed")
			return nil
		}

		if darchive, err = client.Upload(bw.DisplayName(), uint64(dstinfo.Size()), dst); err != nil {
			events <- agentutil.LogError(local, errors.Wrap(err, "archive upload failed"))
			events <- agentutil.LogEvent(local, "deployment failed")
			return err
		}

		return nil
	}, codes.Unavailable)

	if err != nil {
		return err
	}

	events <- agentutil.LogEvent(local, fmt.Sprintf("archive upload completed: who(%s) location(%s)", darchive.Initiator, darchive.Location))

	max := ctx.Concurrency
	if ctx.Concurrency == 0 {
		max = int64(config.Partitioner().Partition(len(c.Members())))
	}

	// only consider the canary node.
	if ctx.Canary {
		peers = agent.NodesToPeers(c.Get(rendezvous.Auto()))
	} else {
		peers = agent.NodesToPeers(c.Members()...)
	}

	peers = deployment.ApplyFilter(ctx.Filter, peers...)
	dopts := agent.DeployOptions{
		Concurrency:       max,
		Timeout:           int64(config.DeployTimeout),
		IgnoreFailures:    ctx.IgnoreFailures,
		SilenceDeployLogs: ctx.SilenceLogs,
	}

	if len(peers) == 0 && !ctx.AllowEmpty {
		cause := errorsx.String("deployment failed, filter did not match any servers")
		events <- agentutil.LogError(local, cause)
		return cause
	}

	events <- agentutil.LogEvent(local, fmt.Sprintf("initiating deploy: concurrency(%d), deployID(%s)", max, bw.RandomID(darchive.DeploymentID)))
	if cause := client.RemoteDeploy(ctx.Context, &dopts, &darchive, peers...); cause != nil {
		events <- agentutil.LogEvent(local, fmt.Sprintln("deployment failed", cause))
	}

	return err
}