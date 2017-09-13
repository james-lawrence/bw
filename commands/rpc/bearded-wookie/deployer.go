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
	"bitbucket.org/jatone/bearded-wookie/archive"
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
		err         error
		dst         *os.File
		c           clustering.Cluster
		creds       credentials.TransportCredentials
		coordinator agent.Client
		port        string
		info        gagent.Archive
	)

	log.Println("deploying", t.deployspace)
	if err = bw.ExpandAndDecodeFile(filepath.Join(bw.LocateDeployspace(bw.DefaultDeployspaceConfigDir), t.environment), &t.config); err != nil {
		return err
	}

	defaults := []clustering.Option{
		clustering.OptionNodeID("deploy"),
		clustering.OptionBindAddress(t.global.systemIP.String()),
		clustering.OptionEventDelegate(eventHandler{}),
	}

	if creds, coordinator, c, err = t.config.Connect(defaults, []clustering.BootstrapOption{}); err != nil {
		return err
	}

	log.Println("connected to cluster cluster")

	if _, port, err = net.SplitHostPort(t.config.Address); err != nil {
		return errors.Wrap(err, "malformed address in configuration")
	}

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

	dst.Seek(0, io.SeekStart)
	if info, err = coordinator.Upload(dst); err != nil {
		return err
	}

	log.Printf("archive created: leader(%s), deployID(%s), location(%s)", info.Leader, bw.RandomID(info.DeploymentID), info.Location)
	_connector := newConnector(port, grpc.WithTransportCredentials(creds))
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

func newConnector(port string, options ...grpc.DialOption) connector {
	return connector{
		port:    port,
		options: options,
	}
}

type connector struct {
	port    string
	options []grpc.DialOption
}

func (t connector) address(peer *memberlist.Node) string {
	return net.JoinHostPort(peer.Addr.String(), t.port)
}

func (t connector) Check2(peer *memberlist.Node) (info gagent.AgentInfo, err error) {
	var (
		c agent.Client
	)
	if c, err = connect(t.address(peer), t.options...); err != nil {
		return info, err
	}
	defer c.Close()

	return c.Info()
}

func (t connector) Check(peer *memberlist.Node) (err error) {
	var (
		info gagent.AgentInfo
	)

	if info, err = t.Check2(peer); err != nil {
		return err
	}

	return deployment.AgentStateToStatus(info.Status)
}

func (t connector) deploy(info gagent.Archive) func(*memberlist.Node) error {
	return func(peer *memberlist.Node) (err error) {
		var (
			c agent.Client
		)
		log.Println("connecting to peer")
		if c, err = connect(t.address(peer), t.options...); err != nil {
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
