package main

import (
	"crypto/tls"
	"log"
	"net"
	"os"
	"path/filepath"
	"time"

	"github.com/james-lawrence/bw"
	"github.com/james-lawrence/bw/agent"
	"github.com/james-lawrence/bw/agent/proxy"
	"github.com/james-lawrence/bw/agent/quorum"
	"github.com/james-lawrence/bw/agentutil"
	"github.com/james-lawrence/bw/cluster"
	"github.com/james-lawrence/bw/clustering"
	"github.com/james-lawrence/bw/clustering/peering"
	"github.com/james-lawrence/bw/clustering/raftutil"
	"github.com/james-lawrence/bw/uploads"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials"
	"google.golang.org/grpc/keepalive"

	"github.com/alecthomas/kingpin"
	"github.com/davecgh/go-spew/spew"
	"github.com/hashicorp/raft"
	"github.com/pkg/errors"
)

type agentCmd struct {
	*global
	config     agent.Config
	configFile string
	server     *grpc.Server
	listener   net.Listener
}

func (t *agentCmd) configure(parent *kingpin.CmdClause) {
	t.cluster.configure(
		parent,
		clusterCmdOptionName(t.config.Name),
		clusterCmdOptionBind(t.config.SWIMBind),
		clusterCmdOptionRaftBind(t.config.RaftBind),
	)

	parent.Flag("agent-name", "name of the node within the network").Default(t.config.Name).StringVar(&t.config.Name)
	parent.Flag("agent-bind", "address for the RPC server to bind to").PlaceHolder(t.config.RPCBind.String()).TCPVar(&t.config.RPCBind)
	parent.Flag("agent-config", "file containing the agent configuration").
		Default(bw.DefaultLocation(bw.DefaultAgentConfig, "")).StringVar(&t.configFile)

	(&directive{agentCmd: t}).configure(parent.Command("directive", "directive based deployment").Default())
	(&dummy{agentCmd: t}).configure(parent.Command("dummy", "dummy deployment"))
}

func (t *agentCmd) bind(aoptions func(*agentutil.Dispatcher, agent.Peer, agent.Config) agent.ServerOption) error {
	var (
		err    error
		c      clustering.Cluster
		creds  *tls.Config
		secret []byte
		p      raftutil.Protocol
		upload uploads.Protocol
	)
	log.SetPrefix("[AGENT] ")

	if err = bw.ExpandAndDecodeFile(t.configFile, &t.config); err != nil {
		return err
	}

	log.Println("configuration:", spew.Sdump(t.config))

	if err = bw.InitializeDeploymentDirectory(t.config.Root); err != nil {
		return err
	}

	if upload, err = t.config.Storage.Protocol(); err != nil {
		return err
	}

	if t.listener, err = net.Listen(t.config.RPCBind.Network(), t.config.RPCBind.String()); err != nil {
		return errors.Wrapf(err, "failed to bind agent to %s", t.config.RPCBind)
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

	local := cluster.NewLocal(t.config.Peer())
	tlscreds := credentials.NewTLS(creds)
	keepalive := grpc.KeepaliveParams(keepalive.ServerParameters{
		Time:    10 * time.Second,
		Timeout: 3 * time.Second,
	})

	t.server = grpc.NewServer(grpc.Creds(tlscreds), keepalive)
	options := []clustering.Option{
		clustering.OptionNodeID(local.Peer.Name),
		clustering.OptionDelegate(local),
		clustering.OptionLogOutput(os.Stderr),
		clustering.OptionSecret(secret),
		clustering.OptionEventDelegate(&p),
		clustering.OptionAliveDelegate(cluster.AliveDefault{}),
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

	cx := cluster.New(local, c)

	dispatcher := agentutil.NewDispatcher(cx, grpc.WithTransportCredentials(tlscreds))
	q := quorum.New(
		cx,
		proxy.NewProxy(cx, dispatcher),
		quorum.OptionUpload(upload),
		quorum.OptionCredentials(tlscreds),
	)
	go (&q).Observe(p, make(chan raft.Observation, 200))

	server := agent.NewServer(
		cx,
		aoptions(dispatcher, local.Peer, t.config),
		agent.ServerOptionShutdown(t.global.shutdown),
	)

	agent.RegisterAgentServer(t.server, server)
	agent.RegisterQuorumServer(t.server, agent.NewQuorum(&q))

	t.runServer(c)

	if err = agentutil.Bootstrap(local.Peer, cx, tlscreds); err != nil {
		return err
	}

	return nil
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
