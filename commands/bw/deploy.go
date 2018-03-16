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
	"time"

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
	"github.com/james-lawrence/bw/x/logx"
	"github.com/pkg/errors"
)

const (
	uxmodeTerm = "term"
	uxmodeLog  = "log"
)

type deployCmd struct {
	global         *global
	uxmode         string
	environment    string
	deployspace    string
	filteredIP     []net.IP
	filteredRegex  []*regexp.Regexp
	ignoreFailures bool
}

func (t *deployCmd) configure(parent *kingpin.CmdClause) {
	common := func(cmd *kingpin.CmdClause) *kingpin.CmdClause {
		cmd.Flag("ux-mode", "choose the user interface").Default(uxmodeLog).EnumVar(&t.uxmode, uxmodeTerm, uxmodeLog)
		cmd.Flag("deployspace", "root directory of the deployspace being deployed").Default(bw.LocateDeployspace(bw.DefaultDeployspaceDir)).StringVar(&t.deployspace)
		cmd.Arg("environment", "the environment configuration to use").Default(bw.DefaultEnvironmentName).StringVar(&t.environment)
		return cmd
	}

	t.deployCmd(common(parent.Command("all", "deploy to all nodes within the cluster").Default()))
	t.localCmd(common(parent.Command("local", "deploy to the local system"))).Hidden()
	t.filteredCmd(common(parent.Command("filtered", "deploy to all the nodes that match one of the provided filters")))
	t.cancelCmd(common(parent.Command("cancel", "cancel any current deploy")))
}

func (t *deployCmd) initializeUX(mode string, events chan agent.Message) {
	t.global.cleanup.Add(1)
	switch mode {
	case uxmodeTerm:
		go ux.NewTermui(t.global.ctx, t.global.shutdown, t.global.cleanup, events)
	default:
		go ux.Logging(t.global.ctx, t.global.cleanup, events)
	}
}

func (t *deployCmd) localCmd(parent *kingpin.CmdClause) *kingpin.CmdClause {
	return parent.Action(t.local)
}

func (t *deployCmd) cancelCmd(parent *kingpin.CmdClause) *kingpin.CmdClause {
	return parent.Action(t.cancel)
}

func (t *deployCmd) deployCmd(parent *kingpin.CmdClause) *kingpin.CmdClause {
	parent.Flag("ignoreFailures", "ignore when an agent fails its deploy").Default("false").BoolVar(&t.ignoreFailures)
	return parent.Action(t.deploy)
}

func (t *deployCmd) filteredCmd(parent *kingpin.CmdClause) *kingpin.CmdClause {
	parent.Flag("name", "regex to match against").RegexpListVar(&t.filteredRegex)
	parent.Flag("ip", "match against the provided IPs").IPListVar(&t.filteredIP)
	parent.Flag("ignoreFailures", "ignore when an agent fails its deploy").Default("false").BoolVar(&t.ignoreFailures)
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

	return t._deploy(deployment.Or(filters...))
}

func (t *deployCmd) deploy(ctx *kingpin.ParseContext) error {
	return t._deploy(deployment.NeverMatch)
}

func (t *deployCmd) _deploy(filter deployment.Filter) error {
	var (
		err     error
		dst     *os.File
		dstinfo os.FileInfo
		client  agent.Client
		config  agent.ConfigClient
		c       clustering.Cluster
		archive agent.Archive
	)

	if config, err = commandutils.LoadConfiguration(t.environment); err != nil {
		return err
	}

	log.Println("configuration:", spew.Sdump(config))
	events := make(chan agent.Message, 100)
	t.initializeUX(t.uxmode, events)

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
			clustering.OptionAliveDelegate(deployclient.AliveHandler{}),
			clustering.OptionLogOutput(ioutil.Discard),
		),
	}

	logRetryError := func(err error) {
		events <- agentutil.LogError(local.Peer, errors.Wrap(err, "connection to cluster failed"))
	}

	events <- agentutil.LogEvent(local.Peer, "connecting to cluster")
	if client, c, err = agent.ConnectClientUntilSuccess(t.global.ctx, config, logRetryError, coptions...); err != nil {
		return err
	}

	events <- agentutil.LogEvent(local.Peer, "connected to cluster")
	go func() {
		<-t.global.ctx.Done()
		if err = client.Close(); err != nil {
			log.Println("failed to close client", err)
		}
	}()

	go func() {
		if watcherr := client.Watch(events); watcherr != nil {
			events <- agentutil.LogError(local.Peer, errors.Wrap(watcherr, "events connection lost"))
		}
		<-t.global.ctx.Done()
		close(events)
	}()

	if err = ioutil.WriteFile(filepath.Join(t.deployspace, bw.EnvFile), []byte(config.Environment), 0600); err != nil {
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

	if err = packer.Pack(dst, t.deployspace); err != nil {
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

	if archive, err = client.Upload(uint64(dstinfo.Size()), dst); err != nil {
		events <- agentutil.LogError(local.Peer, errors.Wrap(err, "archive upload failed"))
		events <- agentutil.LogEvent(local.Peer, "deployment failed")
		return nil
	}

	events <- agentutil.LogEvent(local.Peer, fmt.Sprintf("archive upload completed: location(%s)", archive.Location))

	cx := cluster.New(local, c)
	max := int64(config.Partitioner().Partition(len(cx.Members())))
	peers := deployment.ApplyFilter(filter, cx.Peers()...)
	dopts := agent.DeployOptions{
		Concurrency:    int64(config.Partitioner().Partition(len(cx.Members()))),
		Timeout:        int64(config.DeployTimeout),
		IgnoreFailures: t.ignoreFailures,
	}
	go func() {
		events <- agentutil.LogEvent(local.Peer, fmt.Sprintf("initiating deploy: concurrency(%d), deployID(%s)", max, bw.RandomID(archive.DeploymentID)))

		if cause := client.RemoteDeploy(dopts, archive, peers...); cause != nil {
			events <- agentutil.LogEvent(local.Peer, fmt.Sprintln("deployment failed", cause))
		}
	}()

	return err
}

func (t *deployCmd) cancel(ctx *kingpin.ParseContext) (err error) {
	var (
		client agent.Client
		config agent.ConfigClient
		c      clustering.Cluster
	)
	defer t.global.shutdown()

	if config, err = commandutils.LoadConfiguration(t.environment); err != nil {
		return err
	}

	log.Println("configuration:", spew.Sdump(config))
	events := make(chan agent.Message, 100)
	t.initializeUX(t.uxmode, events)

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
			clustering.OptionAliveDelegate(deployclient.AliveHandler{}),
			clustering.OptionLogOutput(ioutil.Discard),
		),
	}

	logRetryError := func(err error) {
		events <- agentutil.LogError(local.Peer, errors.Wrap(err, "connection to cluster failed"))
	}

	events <- agentutil.LogEvent(local.Peer, "connecting to cluster")
	if client, c, err = agent.ConnectClientUntilSuccess(t.global.ctx, config, logRetryError, coptions...); err != nil {
		return err
	}
	logx.MaybeLog(c.Shutdown())

	events <- agentutil.LogEvent(local.Peer, "connected to cluster")
	go func() {
		<-t.global.ctx.Done()
		if err = client.Close(); err != nil {
			log.Println("failed to close client", err)
		}
	}()

	cmd := agent.DeployCommand{
		Command: agent.DeployCommand_Cancel,
	}

	if err = client.Dispatch(agentutil.DeployCommand(local.Peer, cmd)); err != nil {
		return err
	}

	events <- agentutil.LogEvent(local.Peer, "deploy cancelled")
	time.Sleep(5 * time.Second)
	return nil
}

func (t *deployCmd) local(ctx *kingpin.ParseContext) (err error) {
	var (
		dst     *os.File
		sctx    shell.Context
		dctx    deployment.DeployContext
		root    string
		archive agent.Archive
	)

	local := commandutils.NewClientPeer()
	completed := make(chan deployment.DeployResult)

	if sctx, err = shell.DefaultContext(); err != nil {
		return err
	}

	if root, err = ioutil.TempDir("", "bwlocal"); err != nil {
		return err
	}
	defer os.RemoveAll(root)

	if dst, err = ioutil.TempFile("", "bwarchive"); err != nil {
		return errors.Wrap(err, "archive creation failed")
	}

	defer os.Remove(dst.Name())
	defer dst.Close()

	if err = packer.Pack(dst, t.deployspace); err != nil {
		return errors.Wrap(err, "failed to pack archive")
	}

	if _, err = dst.Seek(0, io.SeekStart); err != nil {
		return errors.WithStack(err)
	}

	if err = packer.Unpack(filepath.Join(root, "archive"), dst); err != nil {
		return errors.Wrap(err, "failed to unpack archive")
	}

	archive = agent.Archive{
		Location: dst.Name(),
	}

	options := []deployment.DeployContextOption{
		deployment.DeployContextOptionCompleted(completed),
	}

	if dctx, err = deployment.NewDeployContext(root, local, archive, options...); err != nil {
		return errors.Wrap(err, "failed to create deployment context")
	}

	deploy := deployment.NewDirective(
		deployment.DirectiveOptionShellContext(sctx),
	)

	deploy.Deploy(dctx)

	result := <-completed
	return result.Error
}
