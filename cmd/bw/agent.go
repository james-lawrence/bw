package main

import (
	"crypto/tls"
	"log"
	"net"
	"path/filepath"

	"github.com/james-lawrence/bw"
	"github.com/james-lawrence/bw/agent"
	"github.com/james-lawrence/bw/certificatecache"
	"github.com/james-lawrence/bw/cluster"
	"github.com/james-lawrence/bw/clustering"
	"github.com/james-lawrence/bw/clustering/peering"
	"github.com/james-lawrence/bw/clustering/raftutil"
	"github.com/james-lawrence/bw/cmd/commandutils"
	"github.com/james-lawrence/bw/daemons"
	"github.com/james-lawrence/bw/deployment"
	"github.com/james-lawrence/bw/storage"

	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials"

	"github.com/alecthomas/kingpin"
	"github.com/davecgh/go-spew/spew"
	"github.com/hashicorp/memberlist"
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
		Default(bw.DefaultLocation(filepath.Join(bw.DefaultEnvironmentName, bw.DefaultAgentConfig), "")).StringVar(&t.configFile)
	(&directive{agentCmd: t}).configure(parent.Command("directive", "directive based deployment").Default())
}

func (t *agentCmd) bind() (err error) {
	var (
		c             clustering.Cluster
		creds         *tls.Config
		keyring       *memberlist.Keyring
		p             raftutil.Protocol
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

	if keyring, err = t.config.Keyring(); err != nil {
		return err
	}

	if creds, err = t.config.BuildServer(); err != nil {
		return err
	}

	local := cluster.NewLocal(t.config.Peer())
	tlscreds := credentials.NewTLS(creds)

	bq := raftutil.BacklogQueue{Backlog: make(chan raftutil.QueuedEvent, 100)}

	cdialer := commandutils.NewClusterDialer(
		t.config,
		clustering.OptionNodeID(local.Peer.Name),
		clustering.OptionDelegate(local),
		clustering.OptionKeyring(keyring),
		clustering.OptionEventDelegate(bq),
		clustering.OptionAliveDelegate(cluster.AliveDefault{}),
		clustering.OptionLogger(commandutils.DebugLog(t.global.debug)),
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

	if tc, err = storage.NewTorrent(cx, opts...); err != nil {
		return err
	}

	go func() {
		for range tdr {
			tcu.ClearTorrents(tc)
		}
	}()

	// go timex.Every(time.Minute, func() {
	// 	log.Println("PID", os.Getpid())
	// 	tcu.PrintTorrentInfo(tc)
	// })

	deployResults = append(deployResults, tdr)

	dctx := daemons.Context{
		Context:        t.global.ctx,
		Shutdown:       t.global.shutdown,
		Cleanup:        t.global.cleanup,
		Upload:         tc.Uploader(),
		Download:       tc.Downloader(),
		RPCCredentials: tlscreds,
		Raft:           p,
		Results:        make(chan deployment.DeployResult, 100),
	}

	if err = daemons.Agent(dctx, cx, t.config); err != nil {
		return err
	}

	if err = daemons.Bootstrap(dctx, cx, t.config); err != nil {
		// if bootstrapping fails shutdown the process.
		return errors.Wrap(err, "failed to bootstrap node shutting down")
	}

	go deployment.ResultBus(dctx.Results, deployResults...)

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
