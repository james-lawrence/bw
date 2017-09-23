package main

import (
	"crypto/tls"
	"log"
	"net"
	"os"
	"path/filepath"
	"time"

	"bitbucket.org/jatone/bearded-wookie"
	"bitbucket.org/jatone/bearded-wookie/agent"
	"bitbucket.org/jatone/bearded-wookie/backoff"
	cp "bitbucket.org/jatone/bearded-wookie/cluster"
	"bitbucket.org/jatone/bearded-wookie/clustering"
	"bitbucket.org/jatone/bearded-wookie/clustering/peering"
	"bitbucket.org/jatone/bearded-wookie/clustering/raftutil"
	gagent "bitbucket.org/jatone/bearded-wookie/deployment/agent"
	"bitbucket.org/jatone/bearded-wookie/x/stringsx"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials"

	"github.com/alecthomas/kingpin"
	"github.com/pkg/errors"
)

type agentCmd struct {
	*global
	config     agent.Config
	configFile string
	network    *net.TCPAddr
	server     *grpc.Server
	listener   net.Listener
}

func (t *agentCmd) configure(parent *kingpin.CmdClause) {
	t.cluster.configure(
		parent,
		clusterCmdOptionBind(
			&net.TCPAddr{
				IP:   t.global.systemIP,
				Port: 2001,
			},
		),
		clusterCmdOptionRaftBind(
			&net.TCPAddr{
				IP:   t.global.systemIP,
				Port: 2002,
			},
		),
	)

	parent.Flag("agent-name", "name of the node within the network").Default(t.config.Name).StringVar(&t.config.Name)
	parent.Flag("agent-bind", "network interface to listen on").Default(t.network.String()).TCPVar(&t.network)
	parent.Flag("agent-config", "file containing the agent configuration").
		Default(bw.DefaultLocation(bw.DefaultAgentConfig, "")).StringVar(&t.configFile)

	(&directive{agentCmd: t}).configure(parent.Command("directive", "directive based deployment").Default())
	(&dummy{agentCmd: t}).configure(parent.Command("dummy", "dummy deployment"))
}

func (t *agentCmd) bind(aoptions func(agent.Config) agent.ServerOption) error {
	var (
		err    error
		c      clustering.Cluster
		creds  *tls.Config
		secret []byte
		p      raftutil.Protocol
	)
	log.SetPrefix("[AGENT] ")

	log.Println("initiated binding rpc server", t.network.String())
	defer log.Println("completed binding rpc server", t.network.String())

	if err = bw.ExpandAndDecodeFile(t.configFile, &t.config); err != nil {
		return err
	}

	log.Printf("configuration: %#v\n", t.config)

	if t.listener, err = net.Listen(t.network.Network(), t.network.String()); err != nil {
		return errors.Wrapf(err, "failed to bind agent to %s", t.network)
	}

	if secret, err = t.config.TLSConfig.Hash(); err != nil {
		return err
	}

	if creds, err = t.config.TLSConfig.BuildServer(); err != nil {
		return err
	}

	if p, err = t.global.cluster.Raft(t.global.ctx, t.config); err != nil {
		return err
	}

	t.server = grpc.NewServer(grpc.Creds(credentials.NewTLS(creds)))
	options := []clustering.Option{
		clustering.OptionNodeID(stringsx.DefaultIfBlank(t.config.Name, t.listener.Addr().String())),
		clustering.OptionDelegate(cp.NewLocal([]byte{})),
		clustering.OptionLogOutput(os.Stderr),
		clustering.OptionSecret(secret),
		clustering.OptionEventDelegate(p),
	}

	fssnapshot := peering.File{
		Path: filepath.Join(t.config.Root, "cluster.snapshot"),
	}

	if c, err = t.global.cluster.Join(fssnapshot, options...); err != nil {
		return errors.Wrap(err, "failed to join cluster")
	}

	t.global.cluster.Snapshot(
		c,
		fssnapshot,
		clustering.SnapshotOptionFrequency(t.config.Cluster.SnapshotFrequency),
		clustering.SnapshotOptionContext(t.global.ctx),
	)

	server := agent.NewServer(
		t.listener.Addr(),
		credentials.NewTLS(creds),
		agent.ComposeServerOptions(aoptions(t.config), agent.ServerOptionCluster(c, secret)),
	)

	agent.RegisterServer(
		t.server,
		server,
	)

	t.runServer(c)
	// go t.bootstrapDeployment(server, c, grpc.WithTransportCredentials(credentials.NewTLS(creds)))
	go p.Overlay(c)

	return nil
}

func (t *agentCmd) bootstrapDeployment(s agent.Server, c clustering.Cluster, doptions ...grpc.DialOption) {
	var (
		err error
	)

	if len(c.Members()) == 1 {
		log.Println("no peers in cluster, skipping bootstrap deployment")
		return
	}

	maybeDeploy := func(latest gagent.Archive, err error) error {
		if err != nil {
			return err
		}
		return s.Deployer.Deploy(&latest)
	}

	b := backoff.Maximum(time.Hour, backoff.Exponential(200*time.Millisecond))

	for attempt := 0; true; attempt++ {
		if err = maybeDeploy(agent.DetermineLatestArchive(s.Address, c, doptions...)); err == nil {
			log.Println("successfully bootstrapped")
			break
		}

		d := b.Backoff(attempt)
		log.Println("failed to determine latest archive, will retry in", d, err)
		time.Sleep(d)
	}
}

func (t *agentCmd) runServer(c clustering.Cluster) {
	go t.server.Serve(t.listener)
	t.global.cleanup.Add(1)
	go func() {
		defer t.global.cleanup.Done()
		<-t.global.ctx.Done()

		log.Println("left cluster", c.Shutdown())
		log.Println("agent shutdown", t.listener.Close())
	}()
}
