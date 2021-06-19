package main

import (
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"net"
	"os"
	"path/filepath"
	"regexp"

	"github.com/alecthomas/kingpin"
	"github.com/logrusorgru/aurora"
	"github.com/manifoldco/promptui"
	"github.com/pkg/errors"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"

	"github.com/james-lawrence/bw"
	"github.com/james-lawrence/bw/agent"
	"github.com/james-lawrence/bw/agent/dialers"
	"github.com/james-lawrence/bw/agentutil"
	packer "github.com/james-lawrence/bw/archive"
	"github.com/james-lawrence/bw/cluster"
	"github.com/james-lawrence/bw/clustering"
	"github.com/james-lawrence/bw/clustering/rendezvous"
	"github.com/james-lawrence/bw/cmd/commandutils"
	"github.com/james-lawrence/bw/daemons"
	"github.com/james-lawrence/bw/deployment"
	"github.com/james-lawrence/bw/directives/shell"
	"github.com/james-lawrence/bw/internal/x/errorsx"
	"github.com/james-lawrence/bw/internal/x/grpcx"
	"github.com/james-lawrence/bw/internal/x/iox"
	"github.com/james-lawrence/bw/internal/x/logx"
	"github.com/james-lawrence/bw/notary"
	"github.com/james-lawrence/bw/ux"
)

type deployCmd struct {
	global         *global
	environment    string
	deploymentID   string
	filteredIP     []net.IP
	filteredRegex  []*regexp.Regexp
	canary         bool
	debug          bool
	ignoreFailures bool
	silenceLogs    bool
}

func (t *deployCmd) configure(parent *kingpin.CmdClause) {
	deployOptions := func(cmd *kingpin.CmdClause) *kingpin.CmdClause {
		cmd.Flag("ignoreFailures", "ignore when an agent fails its deploy").Default("false").BoolVar(&t.ignoreFailures)
		cmd.Flag("silenceLogs", "prevents the logs from being written for a deploy").Default("false").BoolVar(&t.silenceLogs)
		return cmd
	}
	common := func(cmd *kingpin.CmdClause) *kingpin.CmdClause {
		cmd.Arg("environment", "the environment configuration to use").Default(bw.DefaultEnvironmentName).StringVar(&t.environment)
		return cmd
	}

	t.deployCmd(deployOptions(common(parent.Command("default", "deploy to nodes within the cluster").Default())))
	t.redeployCmd(deployOptions(common(parent.Command("archive", "redeploy an archive to nodes within the cluster"))))
	t.localCmd(common(parent.Command("local", "deploy to the local system")))
	t.cancelCmd(common(parent.Command("cancel", "cancel any current deploy")))
}

func (t *deployCmd) initializeUX(c dialers.Defaults, events chan *agent.Message) {
	t.global.cleanup.Add(1)
	go func() {
		ux.Deploy(t.global.ctx, t.global.cleanup, events, ux.OptionFailureDisplay(ux.NewFailureDisplayPrint(c)))
		t.global.shutdown()
	}()
}

func (t *deployCmd) localCmd(parent *kingpin.CmdClause) *kingpin.CmdClause {
	parent.Flag("debug", "leaves artifacts on the filesystem for debugging").BoolVar(&t.debug)
	return parent.Action(t.local)
}

func (t *deployCmd) cancelCmd(parent *kingpin.CmdClause) *kingpin.CmdClause {
	return parent.Action(t.cancel)
}

func (t *deployCmd) deployCmd(parent *kingpin.CmdClause) *kingpin.CmdClause {
	parent.Flag("canary", "deploy only to the canary server - this option is used to consistent select a single server for deployments without having to manually filter").BoolVar(&t.canary)
	parent.Flag("name", "regex to match against").RegexpListVar(&t.filteredRegex)
	parent.Flag("ip", "match against the provided IPs").IPListVar(&t.filteredIP)
	return parent.Action(t.deploy)
}

func (t *deployCmd) redeployCmd(parent *kingpin.CmdClause) *kingpin.CmdClause {
	parent.Flag("canary", "ddeploy only to the canary server - this option is used to consistent select a single server for deployments without having to manually filter").BoolVar(&t.canary)
	parent.Flag("name", "regex to match against").RegexpListVar(&t.filteredRegex)
	parent.Flag("ip", "match against the provided IPs").IPListVar(&t.filteredIP)
	parent.Arg("archive", "deployment ID to redeploy").StringVar(&t.deploymentID)
	return parent.Action(t.redeploy)
}

func (t *deployCmd) deploy(ctx *kingpin.ParseContext) error {
	filters := make([]deployment.Filter, 0, len(t.filteredRegex))
	for _, n := range t.filteredRegex {
		filters = append(filters, deployment.Named(n))
	}

	for _, n := range t.filteredIP {
		filters = append(filters, deployment.IP(n))
	}

	// need a filter to be present for the canary to work.
	if t.canary {
		filters = append(filters, deployment.AlwaysMatch)
	}

	return t._deploy(deployment.Or(filters...), len(filters) == 0)
}

func (t *deployCmd) _deploy(filter deployment.Filter, allowEmpty bool) error {
	var (
		err     error
		dst     *os.File
		dstinfo os.FileInfo
		conn    *grpc.ClientConn
		d       dialers.Defaults
		client  agent.DeployClient
		config  agent.ConfigClient
		c       clustering.LocalRendezvous
		ss      notary.Signer
		archive agent.Archive
		peers   []*agent.Peer
	)

	if ss, err = notary.NewAutoSigner(bw.DisplayName()); err != nil {
		return err
	}

	if config, err = commandutils.LoadConfiguration(t.environment); err != nil {
		return err
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
	if d, c, err = daemons.ConnectClientUntilSuccess(t.global.ctx, config, ss, grpc.WithPerRPCCredentials(ss)); err != nil {
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

	client = agent.NewDeployConn(conn)

	t.initializeUX(d, events)
	events <- agentutil.LogEvent(local, "connected to cluster")

	go agentutil.WatchClusterEvents(t.global.ctx, qd, local, events)

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

	if err = packer.Pack(dst, config.DeployDataDir); err != nil {
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

		if archive, err = client.Upload(bw.DisplayName(), uint64(dstinfo.Size()), dst); err != nil {
			events <- agentutil.LogError(local, errors.Wrap(err, "archive upload failed"))
			events <- agentutil.LogEvent(local, "deployment failed")
			return err
		}

		return nil
	}, codes.Unavailable)

	events <- agentutil.LogEvent(local, fmt.Sprintf("archive upload completed: who(%s) location(%s)", archive.Initiator, archive.Location))

	max := int64(config.Partitioner().Partition(len(c.Members())))

	// only consider the canary node.
	if t.canary {
		peers = agent.NodesToPeers(c.Get(rendezvous.Auto()))
	} else {
		peers = agent.NodesToPeers(c.Members()...)
	}

	peers = deployment.ApplyFilter(filter, peers...)
	dopts := agent.DeployOptions{
		Concurrency:       max,
		Timeout:           int64(config.DeployTimeout),
		IgnoreFailures:    t.ignoreFailures,
		SilenceDeployLogs: t.silenceLogs,
	}

	if len(peers) == 0 && !allowEmpty {
		cause := errorsx.String("deployment failed, filter did not match any servers")
		events <- agentutil.LogError(local, cause)
		return cause
	}

	events <- agentutil.LogEvent(local, fmt.Sprintf("initiating deploy: concurrency(%d), deployID(%s)", max, bw.RandomID(archive.DeploymentID)))
	if cause := client.RemoteDeploy(t.global.ctx, &dopts, &archive, peers...); cause != nil {
		events <- agentutil.LogEvent(local, fmt.Sprintln("deployment failed", cause))
	}

	return err
}

func (t *deployCmd) cancel(ctx *kingpin.ParseContext) (err error) {
	var (
		conn   *grpc.ClientConn
		client agent.DeployClient
		config agent.ConfigClient
		d      dialers.Defaults
		c      clustering.Rendezvous
		ss     notary.Signer
	)

	defer t.global.shutdown()

	if config, err = commandutils.LoadConfiguration(t.environment); err != nil {
		return err
	}

	if ss, err = notary.NewAutoSigner(bw.DisplayName()); err != nil {
		return err
	}

	events := make(chan *agent.Message, 100)

	local := commandutils.NewClientPeer()

	events <- agentutil.LogEvent(local, "connecting to cluster")
	if d, c, err = daemons.ConnectClientUntilSuccess(t.global.ctx, config, ss, grpc.WithPerRPCCredentials(ss)); err != nil {
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

	client = agent.NewDeployConn(conn)

	t.initializeUX(qd, events)

	events <- agentutil.LogEvent(local, "connected to cluster")
	go func() {
		<-t.global.ctx.Done()
		if err = client.Close(); err != nil {
			log.Println("failed to close client", err)
		}
	}()

	cmd := agentutil.DeployCommandCancel(bw.DisplayName())

	if err = client.Cancel(&agent.CancelRequest{Initiator: cmd.Initiator}); err != nil {
		return err
	}

	if err = client.Dispatch(t.global.ctx, agentutil.DeployCommand(local, cmd)); err != nil {
		return err
	}

	events <- agentutil.LogEvent(local, "deploy cancelled")

	return nil
}

func (t *deployCmd) local(ctx *kingpin.ParseContext) (err error) {
	var (
		dst     *os.File
		sctx    shell.Context
		dctx    *deployment.DeployContext
		root    string
		archive agent.Archive
		config  agent.ConfigClient
	)

	if config, err = commandutils.ReadConfiguration(t.environment); err != nil {
		return err
	}

	if err = ioutil.WriteFile(filepath.Join(config.DeployDataDir, bw.EnvFile), []byte(config.Environment), 0600); err != nil {
		return err
	}

	if _, err := os.Stat(filepath.Join(config.Dir(), bw.AuthKeysFile)); !os.IsNotExist(err) {
		if err = iox.Copy(filepath.Join(config.Dir(), bw.AuthKeysFile), filepath.Join(config.DeployDataDir, bw.AuthKeysFile)); err != nil {
			return err
		}
	}

	if err = commandutils.RunLocalDirectives(config); err != nil {
		return errors.Wrap(err, "failed to run local directives")
	}

	local := commandutils.NewClientPeer()

	if sctx, err = shell.DefaultContext(); err != nil {
		return err
	}

	if root, err = ioutil.TempDir("", "bw-local-deploy-*"); err != nil {
		return err
	}

	if t.debug {
		log.Printf("building directory '%s' will remain after exit\n", root)
		defer func() {
			err = errorsx.Compact(err, errorsx.Notification(errors.Errorf("%s build directory '%s' being left on disk", aurora.NewAurora(true).Brown("WARN"), root)))
		}()
	} else {
		defer os.RemoveAll(root)
	}

	if dst, err = ioutil.TempFile("", "bwarchive"); err != nil {
		return errors.Wrap(err, "archive creation failed")
	}

	defer os.Remove(dst.Name())
	defer dst.Close()

	if err = packer.Pack(dst, config.DeployDataDir); err != nil {
		return errors.Wrap(err, "failed to pack archive")
	}

	if _, err = dst.Seek(0, io.SeekStart); err != nil {
		return errors.WithStack(err)
	}

	if err = os.MkdirAll(filepath.Join(root, bw.DirArchive), 0700); err != nil {
		return errors.Wrap(err, "failed to create archive directory")
	}

	if err = packer.Unpack(filepath.Join(root, bw.DirArchive), dst); err != nil {
		return errors.Wrap(err, "failed to unpack archive")
	}

	archive = agent.Archive{
		Location: dst.Name(),
	}

	dopts := agent.DeployOptions{
		Timeout: int64(config.DeployTimeout),
	}

	if dctx, err = deployment.NewRemoteDeployContext(root, local, &dopts, &archive, deployment.DeployContextOptionDisableReset); err != nil {
		return errors.Wrap(err, "failed to create deployment context")
	}

	deploy := deployment.NewDirective(
		deployment.DirectiveOptionShellContext(sctx),
	)
	deploy.Deploy(dctx)

	result := deployment.AwaitDeployResult(dctx)

	return result.Error
}

func (t *deployCmd) redeploy(ctx *kingpin.ParseContext) error {
	filters := make([]deployment.Filter, 0, len(t.filteredRegex))
	for _, n := range t.filteredRegex {
		filters = append(filters, deployment.Named(n))
	}

	for _, n := range t.filteredIP {
		filters = append(filters, deployment.IP(n))
	}

	// need a filter to be present for the canary to work.
	if t.canary {
		filters = append(filters, deployment.AlwaysMatch)
	}

	return t._redeploy(deployment.Or(filters...), len(filters) == 0)
}

func (t *deployCmd) _redeploy(filter deployment.Filter, allowEmpty bool) error {
	var (
		err     error
		conn    *grpc.ClientConn
		d       dialers.Defaults
		client  agent.DeployClient
		config  agent.ConfigClient
		c       clustering.LocalRendezvous
		located agent.Deploy
		archive *agent.Archive
		peers   []*agent.Peer
		ss      notary.Signer
	)

	log.Println("pid", os.Getpid())
	if config, err = commandutils.LoadConfiguration(t.environment); err != nil {
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

	events <- agentutil.LogEvent(local.Peer, "connecting to cluster")
	if d, c, err = daemons.ConnectClientUntilSuccess(t.global.ctx, config, ss, grpc.WithPerRPCCredentials(ss)); err != nil {
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

	client = agent.NewDeployConn(conn)

	t.initializeUX(qd, events)
	events <- agentutil.LogEvent(local.Peer, "connected to cluster")
	go func() {
		<-t.global.ctx.Done()
		if err = client.Close(); err != nil {
			log.Println("failed to close client", err)
		}
	}()

	go agentutil.WatchClusterEvents(t.global.ctx, qd, local.Peer, events)

	cx := cluster.New(local, c)
	if located, err = agentutil.LocateDeployment(cx, qd, agentutil.FilterDeployID(t.deploymentID)); err != nil {
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
	if t.canary {
		peers = agent.NodesToPeers(cx.Get(rendezvous.Auto()))
	} else {
		peers = cx.Peers()
	}

	peers = deployment.ApplyFilter(filter, peers...)
	dopts := agent.DeployOptions{
		Concurrency:       max,
		Timeout:           int64(config.DeployTimeout),
		IgnoreFailures:    t.ignoreFailures,
		SilenceDeployLogs: t.silenceLogs,
	}

	if len(peers) == 0 && !allowEmpty {
		cause := errorsx.String("deployment failed, filter did not match any servers")
		events <- agentutil.LogError(local.Peer, cause)
		return cause
	}

	events <- agentutil.LogEvent(local.Peer, fmt.Sprintf("initiating deploy: concurrency(%d), deployID(%s)", max, bw.RandomID(archive.DeploymentID)))
	if cause := client.RemoteDeploy(t.global.ctx, &dopts, archive, peers...); cause != nil {
		events <- agentutil.LogEvent(local.Peer, fmt.Sprintln("deployment failed", cause))
	}

	return err
}
