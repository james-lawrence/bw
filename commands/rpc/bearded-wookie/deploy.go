package main

import (
	"io"
	"io/ioutil"
	"log"
	"net"
	"os"
	"path/filepath"
	"regexp"

	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials"

	"bitbucket.org/jatone/bearded-wookie"
	"bitbucket.org/jatone/bearded-wookie/agent"
	"bitbucket.org/jatone/bearded-wookie/agentutil"
	"bitbucket.org/jatone/bearded-wookie/archive"
	"bitbucket.org/jatone/bearded-wookie/cluster"
	"bitbucket.org/jatone/bearded-wookie/clustering"
	"bitbucket.org/jatone/bearded-wookie/deployment"
	"bitbucket.org/jatone/bearded-wookie/ux"
	"bitbucket.org/jatone/bearded-wookie/x/systemx"
	"github.com/alecthomas/kingpin"
	"github.com/pkg/errors"
)

type deployCmd struct {
	config        agent.ConfigClient
	global        *global
	environment   string
	deployspace   string
	filteredIP    []net.IP
	filteredRegex []*regexp.Regexp
	doptions      []deployment.Option
}

func (t *deployCmd) configure(parent *kingpin.CmdClause) {
	common := func(cmd *kingpin.CmdClause) *kingpin.CmdClause {
		cmd.Flag("deployspace", "root directory of the deployspace being deployed").Default(bw.LocateDeployspace(bw.DefaultDeployspaceDir)).StringVar(&t.deployspace)
		cmd.Arg("environment", "the environment configuration to use").Default(bw.DefaultEnvironmentName).StringVar(&t.environment)
		return cmd
	}

	t.deployCmd(common(parent.Command("all", "deploy to all nodes within the cluster").Default()))
	t.filteredCmd(common(parent.Command("filtered", "deploy to all the nodes that match one of the provided filters")))
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

	options := append(t.doptions, deployment.DeployOptionFilter(deployment.Or(filters...)))
	return t._deploy(options...)
}

func (t *deployCmd) deploy(ctx *kingpin.ParseContext) error {
	return t._deploy(t.doptions...)
}

func (t *deployCmd) _deploy(options ...deployment.Option) error {
	var (
		err     error
		dst     *os.File
		dstinfo os.FileInfo
		c       clustering.Cluster
		creds   credentials.TransportCredentials
		client  agent.Client
		info    agent.Archive
	)

	local := cluster.NewLocal(
		agent.Peer{
			Name: "deploy",
			Ip:   systemx.HostnameOrLocalhost(),
		},
		cluster.LocalOptionCapability(cluster.NewBitField(cluster.Deploy)),
	)

	log.Println("deploying", t.deployspace)
	coptions := []agent.ConnectOption{
		agent.ConnectOptionConfigPath(filepath.Join(bw.LocateDeployspace(bw.DefaultDeployspaceConfigDir), t.environment)),
		agent.ConnectOptionClustering(
			clustering.OptionDelegate(local),
			clustering.OptionNodeID(local.Peer.Name),
			clustering.OptionBindAddress(local.Peer.Ip),
			clustering.OptionEventDelegate(eventHandler{}),
			clustering.OptionAliveDelegate(cluster.AliveDefault{}),
		),
	}

	if creds, client, c, err = agent.ConnectClient(&t.config, coptions...); err != nil {
		return err
	}

	log.Println("connected to cluster cluster")

	events := make(chan agent.Message, 100)
	go client.Watch(events)
	go ux.Logging(events)
	// go ux.NewTermui(t.global.ctx, t.global.cleanup, events)

	log.Println("uploading archive")
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

	cx := cluster.New(local, c)
	log.Printf("archive created: leader(%s), deployID(%s), location(%s)", info.Peer.Name, bw.RandomID(info.DeploymentID), info.Location)
	_connector := newConnector(grpc.WithTransportCredentials(creds))
	options = append(
		options,
		deployment.DeployOptionChecker(deployment.OperationFunc(_connector.Check)),
		deployment.DeployOptionDeployer(deployment.OperationFunc(_connector.deploy(info))),
	)
	deployment.NewDeploy(
		local.Peer,
		agentutil.NewDispatcher(cx, grpc.WithTransportCredentials(creds)),
		options...,
	).Deploy(cx)

	log.Println("deployment complete")

	// complete.
	t.global.shutdown()

	return err
}

func connect(address string, doptions ...grpc.DialOption) (_czero agent.Client, err error) {
	return agent.Dial(address, doptions...)
}

func newConnector(options ...grpc.DialOption) connector {
	return connector{
		options: options,
	}
}

type connector struct {
	options []grpc.DialOption
}

func (t connector) Check(n agent.Peer) (err error) {
	var (
		c    agent.Client
		info agent.Status
	)

	if c, err = connect(agent.RPCAddress(n), t.options...); err != nil {
		return err
	}

	defer c.Close()

	if info, err = c.Info(); err != nil {
		return err
	}

	return deployment.AgentStateToStatus(info.Peer.Status)
}

func (t connector) deploy(info agent.Archive) func(n agent.Peer) error {
	return func(n agent.Peer) (err error) {
		var (
			c agent.Client
		)

		if c, err = connect(agent.RPCAddress(n), t.options...); err != nil {
			return err
		}
		defer c.Close()

		if err = c.Deploy(info); err != nil {
			return err
		}

		return nil
	}
}
