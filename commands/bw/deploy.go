package main

import (
	"context"
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"net"
	"os"
	"path/filepath"
	"regexp"

	"github.com/alecthomas/kingpin"
	"github.com/davecgh/go-spew/spew"
	"github.com/james-lawrence/bw"
	"github.com/james-lawrence/bw/agent"
	"github.com/james-lawrence/bw/agent/deployclient"
	"github.com/james-lawrence/bw/agentutil"
	packer "github.com/james-lawrence/bw/archive"
	"github.com/james-lawrence/bw/cluster"
	"github.com/james-lawrence/bw/clustering"
	"github.com/james-lawrence/bw/commands/commandutils"
	"github.com/james-lawrence/bw/deployment"
	"github.com/james-lawrence/bw/directives/shell"
	"github.com/james-lawrence/bw/ux"
	"github.com/james-lawrence/bw/x/errorsx"
	"github.com/james-lawrence/bw/x/logx"
	"github.com/pkg/errors"
)

type deployCmd struct {
	global         *global
	environment    string
	filteredIP     []net.IP
	filteredRegex  []*regexp.Regexp
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

	t.deployCmd(deployOptions(common(parent.Command("all", "deploy to all nodes within the cluster").Default())))
	t.filteredCmd(deployOptions(common(parent.Command("filtered", "deploy to all the nodes that match one of the provided filters"))))
	t.localCmd(common(parent.Command("local", "deploy to the local system")))
	t.cancelCmd(common(parent.Command("cancel", "cancel any current deploy")))
}

func (t *deployCmd) initializeUX(d agent.Dialer, events chan agent.Message) {
	t.global.cleanup.Add(1)
	go func() {
		ux.Deploy(t.global.ctx, t.global.cleanup, d, events)
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
	return parent.Action(t.deploy)
}

func (t *deployCmd) filteredCmd(parent *kingpin.CmdClause) *kingpin.CmdClause {
	parent.Flag("name", "regex to match against").RegexpListVar(&t.filteredRegex)
	parent.Flag("ip", "match against the provided IPs").IPListVar(&t.filteredIP)
	return parent.Action(t.filtered)
}

func (t *deployCmd) filtered(ctx *kingpin.ParseContext) error {
	filters := make([]deployment.Filter, 0, len(t.filteredRegex))
	for _, n := range t.filteredRegex {
		filters = append(filters, deployment.Named(n))
	}

	for _, n := range t.filteredIP {
		filters = append(filters, deployment.IP(n))
	}

	return t._deploy(deployment.Or(filters...), false)
}

func (t *deployCmd) deploy(ctx *kingpin.ParseContext) error {
	return t._deploy(deployment.NeverMatch, true)
}

func (t *deployCmd) _deploy(filter deployment.Filter, allowEmpty bool) error {
	var (
		err     error
		dst     *os.File
		dstinfo os.FileInfo
		d       agent.Dialer
		client  agent.Client
		config  agent.ConfigClient
		c       clustering.Cluster
		archive agent.Archive
	)

	if config, err = commandutils.LoadConfiguration(t.environment); err != nil {
		return err
	}

	log.Println("configuration:", spew.Sdump(config))

	if err = commandutils.RunLocalDirectives(config); err != nil {
		return errors.Wrap(err, "failed to run local directives")
	}

	if !commandutils.RemoteTasksAvailable(config) {
		log.Println("no directives to run by the cluster")
		return nil
	}

	events := make(chan agent.Message, 100)

	local := cluster.NewLocal(
		commandutils.NewClientPeer(
			agent.PeerOptionName("local"),
		),
		cluster.LocalOptionCapability(cluster.NewBitField(cluster.Passive)),
	)

	coptions := []agent.ConnectOption{
		agent.ConnectOptionClustering(
			clustering.OptionDelegate(local),
			clustering.OptionNodeID(local.Peer.Name),
			clustering.OptionBindAddress(local.Peer.Ip),
			clustering.OptionEventDelegate(deployclient.NewClusterEventHandler(events)),
			clustering.OptionAliveDelegate(cluster.AliveDefault{}),
			clustering.OptionLogOutput(ioutil.Discard),
		),
	}

	logRetryError := func(err error) {
		events <- agentutil.LogError(local.Peer, errors.Wrap(err, "connection to cluster failed"))
	}

	events <- agentutil.LogEvent(local.Peer, "connecting to cluster")
	if client, d, c, err = agent.ConnectClientUntilSuccess(t.global.ctx, config, logRetryError, coptions...); err != nil {
		return err
	}

	t.initializeUX(d, events)
	events <- agentutil.LogEvent(local.Peer, "connected to cluster")
	go func() {
		<-t.global.ctx.Done()
		if err = client.Close(); err != nil {
			log.Println("failed to close client", err)
		}
	}()

	cx := cluster.New(local, c)
	go agentutil.WatchClusterEvents(t.global.ctx, d, cx, events)

	if err = ioutil.WriteFile(filepath.Join(config.DeployDataDir, bw.EnvFile), []byte(config.Environment), 0600); err != nil {
		return err
	}
	events <- agentutil.LogEvent(local.Peer, "archive upload initiated")

	if dst, err = ioutil.TempFile("", "bwarchive"); err != nil {
		events <- agentutil.LogError(local.Peer, errors.Wrap(err, "archive creation failed"))
		events <- agentutil.LogEvent(local.Peer, "deployment failed")
		return nil
	}
	defer os.Remove(dst.Name())
	defer dst.Close()

	if err = packer.Pack(dst, config.DeployDataDir); err != nil {
		return err
	}

	if dstinfo, err = dst.Stat(); err != nil {
		events <- agentutil.LogError(local.Peer, errors.Wrap(err, "archive creation failed"))
		events <- agentutil.LogEvent(local.Peer, "deployment failed")
		return nil
	}

	if _, err = dst.Seek(0, io.SeekStart); err != nil {
		events <- agentutil.LogError(local.Peer, errors.Wrap(err, "archive creation failed"))
		events <- agentutil.LogEvent(local.Peer, "deployment failed")
		return nil
	}

	if archive, err = client.Upload(DisplayName(), uint64(dstinfo.Size()), dst); err != nil {
		events <- agentutil.LogError(local.Peer, errors.Wrap(err, "archive upload failed"))
		events <- agentutil.LogEvent(local.Peer, "deployment failed")
		return nil
	}

	events <- agentutil.LogEvent(local.Peer, fmt.Sprintf("archive upload completed: location(%s)", archive.Location))

	max := int64(config.Partitioner().Partition(len(cx.Members())))
	peers := deployment.ApplyFilter(filter, cx.Peers()...)
	dopts := agent.DeployOptions{
		Concurrency:       int64(config.Partitioner().Partition(len(cx.Members()))),
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
	if cause := client.RemoteDeploy(dopts, archive, peers...); cause != nil {
		events <- agentutil.LogEvent(local.Peer, fmt.Sprintln("deployment failed", cause))
	}

	return err
}

func (t *deployCmd) cancel(ctx *kingpin.ParseContext) (err error) {
	var (
		client agent.Client
		config agent.ConfigClient
		d      agent.Dialer
		c      clustering.Cluster
	)
	defer t.global.shutdown()

	if config, err = commandutils.LoadConfiguration(t.environment); err != nil {
		return err
	}

	log.Println("configuration:", spew.Sdump(config))

	events := make(chan agent.Message, 100)

	local := cluster.NewLocal(
		commandutils.NewClientPeer(),
		cluster.LocalOptionCapability(cluster.NewBitField(cluster.Passive)),
	)

	coptions := []agent.ConnectOption{
		agent.ConnectOptionClustering(
			clustering.OptionDelegate(local),
			clustering.OptionNodeID(local.Peer.Name),
			clustering.OptionBindAddress(local.Peer.Ip),
			clustering.OptionEventDelegate(deployclient.NewClusterEventHandler(events)),
			clustering.OptionAliveDelegate(cluster.AliveDefault{}),
			clustering.OptionLogOutput(ioutil.Discard),
		),
	}

	logRetryError := func(err error) {
		events <- agentutil.LogError(local.Peer, errors.Wrap(err, "connection to cluster failed"))
	}

	events <- agentutil.LogEvent(local.Peer, "connecting to cluster")
	if client, d, c, err = agent.ConnectClientUntilSuccess(t.global.ctx, config, logRetryError, coptions...); err != nil {
		return err
	}

	t.initializeUX(d, events)
	logx.MaybeLog(c.Shutdown())

	events <- agentutil.LogEvent(local.Peer, "connected to cluster")
	go func() {
		<-t.global.ctx.Done()
		if err = client.Close(); err != nil {
			log.Println("failed to close client", err)
		}
	}()

	cmd := agentutil.DeployCommandCancel(DisplayName())
	if err = client.Dispatch(context.Background(), agentutil.DeployCommand(local.Peer, cmd)); err != nil {
		return err
	}

	if err = client.QuorumCancel(); err != nil {
		return err
	}

	events <- agentutil.LogEvent(local.Peer, "deploy cancelled")

	return nil
}

func (t *deployCmd) local(ctx *kingpin.ParseContext) (err error) {
	var (
		dst     *os.File
		sctx    shell.Context
		dctx    deployment.DeployContext
		root    string
		archive agent.Archive
		config  agent.ConfigClient
	)

	if config, err = commandutils.ReadConfiguration(t.environment); err != nil {
		return err
	}

	log.Println("configuration:", spew.Sdump(config))

	if err = ioutil.WriteFile(filepath.Join(config.DeployDataDir, bw.EnvFile), []byte(config.Environment), 0600); err != nil {
		return err
	}

	if err = commandutils.RunLocalDirectives(config); err != nil {
		return errors.Wrap(err, "failed to run local directives")
	}

	local := commandutils.NewClientPeer()

	if sctx, err = shell.DefaultContext(); err != nil {
		return err
	}

	if root, err = ioutil.TempDir("", "bwlocal"); err != nil {
		return err
	}

	if t.debug {
		log.Println("building in, directory will remain after exit", root)
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

	if err = os.MkdirAll(filepath.Join(root, "archive"), 0700); err != nil {
		return errors.Wrap(err, "failed to create archive directory")
	}

	if err = packer.Unpack(filepath.Join(root, "archive"), dst); err != nil {
		return errors.Wrap(err, "failed to unpack archive")
	}

	archive = agent.Archive{
		Location: dst.Name(),
	}

	dopts := agent.DeployOptions{
		Timeout: int64(config.DeployTimeout),
	}

	if dctx, err = deployment.NewRemoteDeployContext(root, local, dopts, archive); err != nil {
		return errors.Wrap(err, "failed to create deployment context")
	}

	deploy := deployment.NewDirective(
		deployment.DirectiveOptionShellContext(sctx),
	)
	deploy.Deploy(dctx)

	result := deployment.AwaitDeployResult(dctx)
	return result.Error
}
