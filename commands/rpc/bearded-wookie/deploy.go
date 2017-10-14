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

	"bitbucket.org/jatone/bearded-wookie"
	"bitbucket.org/jatone/bearded-wookie/agent"
	"bitbucket.org/jatone/bearded-wookie/agentutil"
	"bitbucket.org/jatone/bearded-wookie/archive"
	"bitbucket.org/jatone/bearded-wookie/cluster"
	"bitbucket.org/jatone/bearded-wookie/clustering"
	"bitbucket.org/jatone/bearded-wookie/deployment"
	"bitbucket.org/jatone/bearded-wookie/ux"
	"bitbucket.org/jatone/bearded-wookie/x/debugx"
	"bitbucket.org/jatone/bearded-wookie/x/systemx"
	"github.com/alecthomas/kingpin"
	"github.com/pkg/errors"
)

const (
	uxmodeTerm = "term"
	uxmodeLog  = "log"
)

type deployCmd struct {
	config        agent.ConfigClient
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
	switch mode {
	case uxmodeTerm:
		t.global.cleanup.Add(1)
		go ux.NewTermui(t.global.ctx, t.global.shutdown, t.global.cleanup, events)
	default:
		go ux.Logging(events)
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
		c       clustering.Cluster
		info    agent.Archive
	)

	local := cluster.NewLocal(
		agent.Peer{
			Name: "deploy",
			Ip:   systemx.HostnameOrLocalhost(),
		},
		cluster.LocalOptionCapability(cluster.NewBitField(cluster.Deploy)),
	)

	coptions := []agent.ConnectOption{
		agent.ConnectOptionConfigPath(filepath.Join(bw.LocateDeployspace(bw.DefaultDeployspaceConfigDir), t.environment)),
		agent.ConnectOptionClustering(
			clustering.OptionDelegate(local),
			clustering.OptionNodeID(local.Peer.Name),
			clustering.OptionBindAddress(local.Peer.Ip),
			clustering.OptionEventDelegate(cluster.LoggingEventHandler{}),
			clustering.OptionAliveDelegate(cluster.AliveDefault{}),
		),
	}

	if _, client, c, err = agent.ConnectClient(&t.config, coptions...); err != nil {
		return err
	}

	go func() {
		<-t.global.ctx.Done()
		if err = client.Close(); err != nil {
			log.Println("failed to close client")
		}
	}()

	log.Println("connected to cluster")
	debugx.Printf("configuration:\n%#v\n", t.config)

	events := make(chan agent.Message, 100)
	go client.Watch(events)
	t.initializeUX(t.uxmode, events)

	events <- agentutil.LogEvent(local.Peer, "uploading archive")

	if dst, err = ioutil.TempFile("", "bwarchive"); err != nil {
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
	max := int64(t.config.Partitioner().Partition(len(cx.Members())))

	go func() {
		defer t.global.shutdown()
		defer time.Sleep(3 * time.Second)

		if cause := client.RemoteDeploy(max, info, deployment.ApplyFilter(filter, cx.Peers()...)...); cause != nil {
			log.Println("deployment failed", cause)
		}

		log.Println("deployment complete")
	}()

	return err
}
