package main

import (
	"io"
	"io/ioutil"
	"log"
	"net"
	"os"
	"path/filepath"
	"regexp"
	"time"

	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials"

	"bitbucket.org/jatone/bearded-wookie"
	"bitbucket.org/jatone/bearded-wookie/agent"
	"bitbucket.org/jatone/bearded-wookie/agentutil"
	"bitbucket.org/jatone/bearded-wookie/archive"
	"bitbucket.org/jatone/bearded-wookie/cluster"
	"bitbucket.org/jatone/bearded-wookie/clustering"
	"bitbucket.org/jatone/bearded-wookie/deployment"
	gagent "bitbucket.org/jatone/bearded-wookie/deployment/agent"
	"bitbucket.org/jatone/bearded-wookie/ux"
	"github.com/alecthomas/kingpin"
	"github.com/hashicorp/memberlist"
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
		info    gagent.Archive
	)

	local := cluster.NewLocal(
		gagent.Peer{
			Name: "deploy",
			Ip:   t.global.systemIP.String(),
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
		),
	}

	if creds, client, c, err = agent.ConnectClient(&t.config, coptions...); err != nil {
		return err
	}

	log.Println("connected to cluster cluster")

	events := make(chan gagent.Message, 100)
	go client.Watch(events)
	go func() {
		for m := range events {
			switch m.Type {
			default:
				log.Printf("%s - %s: \n", time.Unix(m.GetTs(), 0).Format(time.Stamp), m.Type)
			}
		}
	}()
	// # TODO connect to event stream.

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

	log.Printf("archive created: leader(%s), deployID(%s), location(%s)", info.Peer.Name, bw.RandomID(info.DeploymentID), info.Location)
	_connector := newConnector(grpc.WithTransportCredentials(creds))
	options = append(
		options,
		deployment.DeployOptionChecker(_connector),
		deployment.DeployOptionDeployer(_connector.deploy(info)),
	)
	deployment.NewDeploy(
		// ux.NewTermui(t.global.cleanup, t.global.ctx),
		ux.Logging(),
		options...,
	).Deploy(c)

	log.Println("deployment complete")

	// complete.
	t.global.shutdown()

	return err
}

func connect(address string, doptions ...grpc.DialOption) (_czero agent.Client, err error) {
	return agent.DialClient(address, doptions...)
}

func newConnector(options ...grpc.DialOption) connector {
	return connector{
		options: options,
	}
}

type connector struct {
	port    string
	options []grpc.DialOption
}

func (t connector) Check2(n *memberlist.Node) (info gagent.Status, err error) {
	var (
		c agent.Client
	)
	if c, err = connect(agentutil.NodeRPCAddress(n), t.options...); err != nil {
		return info, err
	}
	defer c.Close()

	return c.Info()
}

func (t connector) Check(peer *memberlist.Node) (err error) {
	var (
		info gagent.Status
	)

	if info, err = t.Check2(peer); err != nil {
		return err
	}

	return deployment.AgentStateToStatus(info.Peer.Status)
}

func (t connector) deploy(info gagent.Archive) func(*memberlist.Node) error {
	return func(n *memberlist.Node) (err error) {
		var (
			c agent.Client
		)

		log.Println("connecting to peer")
		if c, err = connect(agentutil.NodeRPCAddress(n), t.options...); err != nil {
			return err
		}
		defer c.Close()

		log.Println("deploying to peer")
		if err = c.Deploy(info); err != nil {
			return err
		}

		return nil
	}
}
