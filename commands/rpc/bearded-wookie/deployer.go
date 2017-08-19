package main

import (
	"encoding/hex"
	"io"
	"io/ioutil"
	"log"
	"net"
	"os"
	"regexp"

	"google.golang.org/grpc"

	"bitbucket.org/jatone/bearded-wookie/agent"
	"bitbucket.org/jatone/bearded-wookie/archive"
	cp "bitbucket.org/jatone/bearded-wookie/cluster"
	"bitbucket.org/jatone/bearded-wookie/cluster/serfdom"
	"bitbucket.org/jatone/bearded-wookie/clustering"
	"bitbucket.org/jatone/bearded-wookie/deployment"
	gagent "bitbucket.org/jatone/bearded-wookie/deployment/agent"
	"bitbucket.org/jatone/bearded-wookie/ux"

	"github.com/alecthomas/kingpin"
	"github.com/hashicorp/memberlist"
)

type deployCmd struct {
	*global
	environment   string
	workspace     string
	filteredIP    []net.IP
	filteredRegex []*regexp.Regexp
	doptions      []deployment.Option
}

func (t *deployCmd) configure(parent *kingpin.CmdClause) {
	t.global.cluster.configure(
		parent,
		clusterCmdOptionName("deploy"),
		clusterCmdOptionBind(
			&net.TCPAddr{
				IP:   t.global.systemIP,
				Port: 0,
			},
		),
		clusterCmdOptionMinPeers(1),
	)

	parent.Flag("workspace", "root directory of the workspace to be deployed").Default(uploadArchiveRootDefault).StringVar(&t.workspace)
	parent.Arg("environment", "the environment").StringVar(&t.environment)
}

func (t *deployCmd) deployCmd(parent *kingpin.CmdClause) {
	t.configure(parent)
	parent.Action(t.deploy)
}

func (t *deployCmd) filteredCmd(parent *kingpin.CmdClause) {
	cmd := parent.Command("deploy", "deploy to and nodes that match one of the provided filters")
	t.configure(cmd)
	cmd.Flag("name", "regex to match against").RegexpListVar(&t.filteredRegex)
	cmd.Flag("ip", "match against the provided IPs").IPListVar(&t.filteredIP)
	cmd.Action(t.filtered)
}

func (t *deployCmd) filtered(ctx *kingpin.ParseContext) error {
	filters := make([]deployment.Filter, 0, len(t.filteredRegex))
	for _, n := range t.filteredRegex {
		filters = append(filters, deployment.Named(n))
	}
	for _, n := range t.filteredIP {
		filters = append(filters, deployment.IP(n))
	}

	return t._deploy(
		append(
			t.doptions,
			deployment.DeployOptionFilter(deployment.Or(filters...)),
		)...,
	)
}

func (t *deployCmd) deploy(ctx *kingpin.ParseContext) error {
	return t._deploy(t.doptions...)
}

func (t *deployCmd) _deploy(options ...deployment.Option) error {
	var (
		err         error
		dst         *os.File
		c           clustering.Cluster
		coordinator agent.Client
		info        gagent.Archive
	)

	coptions := []clustering.Option{
		clustering.OptionDelegate(serfdom.NewLocal(cp.BitFieldMerge([]byte(nil), cp.Lurker))),
		clustering.OptionLogger(os.Stderr),
	}
	if c, err = t.global.cluster.Join(coptions...); err != nil {
		return err
	}

	if coordinator, err = connectToCoordinator(c); err != nil {
		return err
	}

	// # TODO connect to message stream.

	log.Println("uploading archive")
	if dst, err = ioutil.TempFile("", "bwarchive"); err != nil {
		return err
	}
	defer os.Remove(dst.Name())
	defer dst.Close()

	if err = archive.Pack(dst, t.workspace); err != nil {
		return err
	}

	dst.Seek(0, io.SeekStart)
	if info, err = coordinator.Upload(dst); err != nil {
		return err
	}

	log.Printf("archive created: leader(%s), deployID(%s), location(%s)", info.Leader, hex.EncodeToString(info.DeploymentID), info.Location)

	deployment.NewDeploy(
		// ux.NewTermui(t.global.cleanup, t.global.ctx),
		ux.Logging(),
		options...,
	).Deploy(c)

	log.Println("deployment complete")

	// complete.
	t.shutdown()

	return err
}

func connectToCoordinator(c clustering.Cluster) (_czero agent.Client, err error) {
	var (
		randomness  []byte
		coordinator agent.Client
	)

	if randomness, err = agent.GenerateID(); err != nil {
		return _czero, err
	}

	if coordinator, err = connect(c.Get(randomness)); err != nil {
		return _czero, err
	}

	return coordinator, err
}

func connect(peer *memberlist.Node) (_czero agent.Client, err error) {
	address := net.JoinHostPort(peer.Addr.String(), "2000")
	doptions := []grpc.DialOption{
		grpc.WithInsecure(),
		// grpc.WithCompressor(grpc.NewGZIPCompressor()),
		// grpc.WithDecompressor(grpc.NewGZIPDecompressor()),
	}

	return agent.DialClient(address, doptions...)
}

type status struct{}

func (status) Check(peer *memberlist.Node) (err error) {
	var (
		c    agent.Client
		info gagent.AgentInfo
	)
	if c, err = connect(peer); err != nil {
		return err
	}
	defer c.Close()

	if info, err = c.Info(); err != nil {
		return err
	}

	return deployment.AgentStateToStatus(info.Status)
}

func deploy(info gagent.Archive) func(*memberlist.Node) error {
	return func(peer *memberlist.Node) (err error) {
		var (
			c agent.Client
		)
		log.Println("connecting to peer")
		if c, err = connect(peer); err != nil {
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
