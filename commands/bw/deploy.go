package main

import (
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"net"
	"os"
	"regexp"

	"github.com/alecthomas/kingpin"
	"github.com/davecgh/go-spew/spew"
	"github.com/james-lawrence/bw"
	"github.com/james-lawrence/bw/agent"
	"github.com/james-lawrence/bw/agentutil"
	"github.com/james-lawrence/bw/archive"
	"github.com/james-lawrence/bw/cluster"
	"github.com/james-lawrence/bw/clustering"
	"github.com/james-lawrence/bw/deployment"
	"github.com/james-lawrence/bw/ux"
	"github.com/james-lawrence/bw/x/debugx"
	"github.com/james-lawrence/bw/x/systemx"
	"github.com/pkg/errors"
)

const (
	uxmodeTerm = "term"
	uxmodeLog  = "log"
)

type deployCmd struct {
	global        *global
	uxmode        string
	environment   string
	deployspace   string
	filteredIP    []net.IP
	filteredRegex []*regexp.Regexp
}

func (t *deployCmd) configure(parent *kingpin.CmdClause) {
	common := func(cmd *kingpin.CmdClause) *kingpin.CmdClause {
		cmd.Flag("ux-mode", "choose the user interface").Default(uxmodeTerm).EnumVar(&t.uxmode, uxmodeTerm, uxmodeLog)
		cmd.Flag("deployspace", "root directory of the deployspace being deployed").Default(bw.LocateDeployspace(bw.DefaultDeployspaceDir)).StringVar(&t.deployspace)
		cmd.Arg("environment", "the environment configuration to use").Default(bw.DefaultEnvironmentName).StringVar(&t.environment)
		return cmd
	}

	t.deployCmd(common(parent.Command("all", "deploy to all nodes within the cluster").Default()))
	t.filteredCmd(common(parent.Command("filtered", "deploy to all the nodes that match one of the provided filters")))
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
		info    agent.Archive
	)

	if config, err = loadConfiguration(t.environment); err != nil {
		return err
	}

	log.Println("configuration:", spew.Sdump(config))

	local := cluster.NewLocal(
		agent.Peer{
			Name: "deploy",
			Ip:   systemx.HostnameOrLocalhost(),
		},
		cluster.LocalOptionCapability(cluster.NewBitField(cluster.Deploy)),
	)

	coptions := []agent.ConnectOption{
		agent.ConnectOptionClustering(
			clustering.OptionDelegate(local),
			clustering.OptionNodeID(local.Peer.Name),
			clustering.OptionBindAddress(local.Peer.Ip),
			clustering.OptionEventDelegate(cluster.LoggingEventHandler{}),
			clustering.OptionAliveDelegate(cluster.AliveDefault{}),
		),
	}

	if _, client, c, err = agent.ConnectClient(config, coptions...); err != nil {
		return err
	}

	go func() {
		<-t.global.ctx.Done()
		if err = client.Close(); err != nil {
			log.Println("failed to close client")
		}
	}()

	log.Println("connected to cluster")
	debugx.Printf("configuration:\n%#v\n", config)

	events := make(chan agent.Message, 100)
	go func() {
		if watcherr := client.Watch(events); watcherr != nil {
			events <- agentutil.LogEvent(local.Peer, fmt.Sprintf("event watch failed: %v", watcherr))
		}
		<-t.global.ctx.Done()
		close(events)
	}()

	t.initializeUX(t.uxmode, events)

	events <- agentutil.LogEvent(local.Peer, "uploading archive")

	if dst, err = ioutil.TempFile("", "bwarchive"); err != nil {
		log.Println("failed to build archive file", err)
		return err
	}
	defer os.Remove(dst.Name())
	defer dst.Close()

	if err = archive.Pack(dst, t.deployspace); err != nil {
		return err
	}

	if dstinfo, err = dst.Stat(); err != nil {
		return errors.WithStack(err)
	}

	if _, err = dst.Seek(0, io.SeekStart); err != nil {
		return errors.WithStack(err)
	}

	if info, err = client.Upload(uint64(dstinfo.Size()), dst); err != nil {
		return err
	}

	events <- agentutil.LogEvent(local.Peer, fmt.Sprintf("archive created: leader(%s), deployID(%s), location(%s)", info.Peer.Name, bw.RandomID(info.DeploymentID), info.Location))

	cx := cluster.New(local, c)
	max := int64(config.Partitioner().Partition(len(cx.Members())))
	peers := deployment.ApplyFilter(filter, cx.Peers()...)
	go func() {
		events <- agentutil.LogEvent(local.Peer, fmt.Sprintf("initiating deploy: concurrency(%d), deployID(%s), total peers(%d)", max, bw.RandomID(info.DeploymentID), len(peers)))

		if cause := client.RemoteDeploy(max, info, peers...); cause != nil {
			events <- agentutil.LogEvent(local.Peer, fmt.Sprintln("deployment failed", cause))
		}

		log.Println("deployment complete")
	}()

	return err
}
