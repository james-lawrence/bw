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
	"github.com/james-lawrence/bw/agent/observers"
	"github.com/james-lawrence/bw/agent/proxy"
	"github.com/james-lawrence/bw/agent/quorum"
	"github.com/james-lawrence/bw/agentutil"
	"github.com/james-lawrence/bw/bootstrap"
	"github.com/james-lawrence/bw/certificatecache"
	"github.com/james-lawrence/bw/cluster"
	"github.com/james-lawrence/bw/clustering"
	"github.com/james-lawrence/bw/clustering/peering"
	"github.com/james-lawrence/bw/clustering/raftutil"
	"github.com/james-lawrence/bw/commands/commandutils"
	"github.com/james-lawrence/bw/deployment"
	"github.com/james-lawrence/bw/storage"

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
}

func (t *agentCmd) configure(parent *kingpin.CmdClause) {
	t.global.cluster.configure(parent, &t.config)

	parent.Flag("agent-name", "name of the node within the network").Default(t.config.Name).StringVar(&t.config.Name)
	parent.Flag("agent-bind", "address for the RPC server to bind to").PlaceHolder(t.config.RPCBind.String()).TCPVar(&t.config.RPCBind)
	parent.Flag("agent-torrent", "address for the Torrent server to bind to").PlaceHolder(t.config.TorrentBind.String()).TCPVar(&t.config.TorrentBind)
	parent.Flag("agent-config", "file containing the agent configuration").
		Default(bw.DefaultLocation(bw.DefaultAgentConfig, "")).StringVar(&t.configFile)
	(&directive{agentCmd: t}).configure(parent.Command("directive", "directive based deployment").Default())
}

func (t *agentCmd) bind(newCoordinator func(agentContext, storage.DownloadProtocol) deployment.Coordinator) error {
	var (
		err           error
		observersdir  observers.Directory
		rpcBind       net.Listener
		server        *grpc.Server
		c             clustering.Cluster
		creds         *tls.Config
		secret        []byte
		p             raftutil.Protocol
		upload        storage.UploadProtocol
		download      storage.DownloadProtocol
		deployResults []chan deployment.DeployResult
	)

	log.SetPrefix("[AGENT] ")

	if err = bw.ExpandAndDecodeFile(t.configFile, &t.config); err != nil {
		return err
	}

	log.Println("configuration:", spew.Sdump(t.config))

	if err = bw.InitializeDeploymentDirectory(t.config.Root); err != nil {
		return err
	}

	if err = certificatecache.FromConfig(t.config.CredentialsDir, t.config.CredentialsMode, t.configFile); err != nil {
		return err
	}

	if rpcBind, err = net.Listen(t.config.RPCBind.Network(), t.config.RPCBind.String()); err != nil {
		return errors.Wrapf(err, "failed to bind agent to %s", t.config.RPCBind)
	}

	if secret, err = t.config.Hash(); err != nil {
		return err
	}

	if creds, err = t.config.BuildServer(); err != nil {
		return err
	}

	if observersdir, err = observers.NewDirectory(filepath.Join(t.config.Root, "observers")); err != nil {
		return err
	}

	local := cluster.NewLocal(t.config.Peer())
	tlscreds := credentials.NewTLS(creds)
	keepalive := grpc.KeepaliveParams(keepalive.ServerParameters{
		MaxConnectionIdle: 1 * time.Hour,
		Time:              1 * time.Minute,
		Timeout:           2 * time.Minute,
	})
	bq := raftutil.BacklogQueue{Backlog: make(chan raftutil.QueuedEvent, 100)}
	cdialer := commandutils.NewClusterDialer(
		t.config,
		clustering.OptionNodeID(local.Peer.Name),
		clustering.OptionDelegate(local),
		clustering.OptionLogOutput(os.Stderr),
		clustering.OptionSecret(secret),
		clustering.OptionEventDelegate(bq),
		clustering.OptionAliveDelegate(cluster.AliveDefault{}),
	)
	fssnapshot := peering.File{
		Path: filepath.Join(t.config.Root, "cluster.snapshot"),
	}

	if c, err = t.global.cluster.Join(t.global.ctx, t.config, cdialer, fssnapshot); err != nil {
		return errors.Wrap(err, "failed to join cluster")
	}

	t.global.cluster.Snapshot(
		c,
		fssnapshot,
		clustering.SnapshotOptionFrequency(t.config.SnapshotFrequency),
		clustering.SnapshotOptionContext(t.global.ctx),
	)

	sq := raftutil.BacklogQueueWorker{
		Provider: cluster.NewRaftAddressProvider(c),
		Queue:    make(chan raftutil.Event, 100),
	}
	go sq.Background(bq)
	if p, err = t.global.cluster.Raft(t.global.ctx, t.config, sq); err != nil {
		return err
	}

	cx := cluster.New(local, c)
	var (
		tc  storage.TorrentConfig
		tcu storage.TorrentUtil
		tdr = make(chan deployment.DeployResult)
	)

	opts := []storage.TorrentOption{
		storage.TorrentOptionBind(t.config.TorrentBind),
		storage.TorrentOptionDHTPeers(cx),
		storage.TorrentOptionDataDir(filepath.Join(t.config.Root, bw.DirTorrents)),
	}

	if tc, err = storage.NewTorrent(opts...); err != nil {
		return err
	}

	go func() {
		for range tdr {
			tcu.ClearTorrents(tc)
			// tcu.PrintTorrentInfo(tc)
		}
	}()

	deployResults = append(deployResults, tdr)
	upload, download = tc.Uploader(), tc.Downloader()

	dialer := agent.NewDialer(agent.DefaultDialerOptions(grpc.WithTransportCredentials(tlscreds))...)
	qdialer := agent.NewQuorumDialer(dialer)
	dispatcher := agentutil.NewDispatcher(cx, qdialer)
	actx := agentContext{Dispatcher: dispatcher, Config: t.config, completedDeploys: make(chan deployment.DeployResult, 100)}
	coordinator := newCoordinator(actx, download)

	q := quorum.New(
		observersdir,
		cx,
		proxy.NewProxy(cx),
		upload,
		quorum.OptionDialer(dialer),
	)
	go (&q).Observe(p, make(chan raft.Observation, 200))

	a := agent.NewServer(
		cx,
		agent.ServerOptionDeployer(&coordinator),
		agent.ServerOptionShutdown(t.global.shutdown),
	)

	aq := agent.NewQuorum(&q)
	server = grpc.NewServer(grpc.Creds(tlscreds), keepalive)
	agent.RegisterAgentServer(server, a)
	agent.RegisterQuorumServer(server, aq)
	t.runServer(server, rpcBind)
	t.gracefulShutdown(c, rpcBind)

	if !bootstrap.UntilSuccess(t.config.BootstrapAttempts, local.Peer, cx, dialer, coordinator) {
		// if bootstrapping fails shutdown the process.
		return errors.New("failed to bootstrap node shutting down")
	}

	go deployment.ResultBus(actx.completedDeploys, deployResults...)
	return nil
}

func (t *agentCmd) runServer(server *grpc.Server, listeners ...net.Listener) {
	for _, l := range listeners {
		go server.Serve(l)
	}
}

func (t *agentCmd) gracefulShutdown(c clustering.Cluster, listeners ...net.Listener) {
	t.global.cleanup.Add(1)
	go func() {
		defer t.global.cleanup.Done()
		<-t.global.ctx.Done()

		log.Println("left cluster", c.Shutdown())
		for _, l := range listeners {
			log.Println("agent shutdown", l.Close())
		}
	}()
}
