package main

import (
	"log"
	"net"
	"os"

	"bitbucket.org/jatone/bearded-wookie"
	"bitbucket.org/jatone/bearded-wookie/agent"
	cp "bitbucket.org/jatone/bearded-wookie/cluster"
	"bitbucket.org/jatone/bearded-wookie/clustering"
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
	)

	parent.Flag("agent-bind", "network interface to listen on").Default(t.network.String()).TCPVar(&t.network)
	parent.Flag("agent-config", "file containing the agent configuration").
		Default(bw.DefaultLocation(bw.DefaultAgentConfig, "")).StringVar(&t.configFile)

	(&directive{agentCmd: t}).configure(parent.Command("directive", "directive based deployment").Default())
	(&dummy{agentCmd: t}).configure(parent.Command("dummy", "dummy deployment"))
}

func (t *agentCmd) bind(addr net.Addr, aoptions func(agent.Config) agent.ServerOption) error {
	var (
		err    error
		c      clustering.Cluster
		creds  credentials.TransportCredentials
		secret []byte
	)
	log.SetPrefix("[AGENT] ")

	log.Println("initiated binding rpc server", t.network.String())
	defer log.Println("completed binding rpc server", t.network.String())

	if err = bw.ExpandAndDecodeFile(t.configFile, &t.config); err != nil {
		return err
	}

	log.Printf("configuration: %#v\n", t.config)

	if t.listener, err = net.Listen("tcp", t.network.String()); err != nil {
		return errors.Wrapf(err, "failed to bind agent to %s", t.network)
	}

	if secret, err = t.config.TLSConfig.Hash(); err != nil {
		return err
	}

	if creds, err = t.config.TLSConfig.BuildServer(); err != nil {
		return err
	}

	t.server = grpc.NewServer(grpc.Creds(creds))
	options := []clustering.Option{
		clustering.OptionDelegate(cp.NewLocal([]byte{})),
		clustering.OptionLogger(os.Stderr),
		clustering.OptionSecret(secret),
	}

	if c, err = t.global.cluster.Join(options...); err != nil {
		return errors.Wrap(err, "failed to join cluster")
	}

	agent.RegisterServer(
		t.server,
		agent.NewServer(addr, agent.ComposeServerOptions(aoptions(t.config), agent.ServerOptionCluster(c, secret))),
	)

	go t.server.Serve(t.listener)
	t.global.cleanup.Add(1)
	go func() {
		defer t.global.cleanup.Done()
		<-t.global.ctx.Done()

		log.Println("left cluster", c.Shutdown())
		log.Println("agent shutdown", t.listener.Close())
	}()

	return nil
}
