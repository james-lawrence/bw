package main

import (
	"crypto/tls"
	"fmt"
	"log"
	"path/filepath"
	"time"

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
	"github.com/james-lawrence/bw/notary"
	"github.com/james-lawrence/bw/storage"

	"google.golang.org/grpc/keepalive"
	"github.com/golang/protobuf/proto"
	"github.com/alecthomas/kingpin"
	"github.com/davecgh/go-spew/spew"
	"github.com/hashicorp/memberlist"
	"github.com/hashicorp/raft"
	"github.com/pkg/errors"
)

type agentCmd struct {
	*global
	config     agent.Config
	configFile string
	raftFile string
	raftVerbose bool
}

func (t *agentCmd) configure(parent *kingpin.CmdClause) {
	t.global.cluster.configure(parent, &t.config)

	parent.Flag("agent-name", "name of the node within the network").Default(t.config.Name).StringVar(&t.config.Name)
	parent.Flag("agent-bind", "address for the RPC server to bind to").PlaceHolder(
		t.config.RPCBind.String(),
	).Envar(
		bw.EnvAgentRPCBind,
	).TCPVar(&t.config.RPCBind)
	parent.Flag("agent-discovery", "address for the discovery server to bind to").PlaceHolder(
		t.config.DiscoveryBind.String(),
	).Envar(
		bw.EnvAgentDiscoveryBind,
	).TCPVar(&t.config.DiscoveryBind)
	parent.Flag("agent-torrent", "address for the torrent server to bind to").PlaceHolder(
		t.config.TorrentBind.String(),
	).Envar(
		bw.EnvAgentTorrentBind,
	).TCPVar(&t.config.TorrentBind)
	parent.Flag("agent-autocert", "address for the autocert server to bind to").PlaceHolder(
		t.config.AutocertBind.String(),
	).Envar(
		bw.EnvAgentAutocertBind,
	).TCPVar(&t.config.AutocertBind)
	parent.Flag("agent-config", "file containing the agent configuration").
		Default(bw.DefaultLocation(filepath.Join(bw.DefaultEnvironmentName, bw.DefaultAgentConfig), "")).StringVar(&t.configFile)

	(&directive{agentCmd: t}).configure(parent.Command("directive", "directive based deployment").Default())

	t.displayCmd(parent.Command("quorum-state", "display the quorum state, only can be run on the server"))
}

func (t *agentCmd) bind() (err error) {
	var (
		c             clustering.Cluster
		tlscreds      *tls.Config
		keyring       *memberlist.Keyring
		p             raftutil.Protocol
		deployResults []chan deployment.DeployResult
		ns            notary.Composite
	)

	log.SetPrefix(fmt.Sprintf("[AGENT - %s] ", t.config.Name))

	if t.config, err = commandutils.LoadAgentConfig(t.configFile, t.config); err != nil {
		return err
	}

	log.Println("configuration:", spew.Sdump(t.config))

	if err = bw.InitializeDeploymentDirectory(t.config.Root); err != nil {
		return err
	}

	if keyring, err = t.config.Keyring(); err != nil {
		return err
	}

	if err = certificatecache.AutomaticTLSAgent(t.config.ServerName, t.config.CredentialsDir); err != nil {
		return err
	}

	if ns, err = notary.NewFromFile(filepath.Join(t.config.Root, bw.DirAuthorizations), t.configFile); err != nil {
		return err
	}

	local := cluster.NewLocal(t.config.Peer())
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

	if tlscreds, err = daemons.TLSGenServer(t.config); err != nil {
		return err
	}

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

	deployResults = append(deployResults, tdr)

	dctx := daemons.Context{
		ConfigurationFile: t.configFile,
		Config:            t.config,
		Context:           t.global.ctx,
		Shutdown:          t.global.shutdown,
		Cleanup:           t.global.cleanup,
		Upload:            tc.Uploader(),
		Download:          tc.Downloader(),
		NotaryStorage:     ns,
		RPCCredentials:    tlscreds,
		RPCKeepalive: keepalive.ServerParameters{
			MaxConnectionIdle: 1 * time.Hour,
			Time:              1 * time.Minute,
			Timeout:           2 * time.Minute,
		},
		Cluster: cx,
		Raft:    p,
		Results: make(chan deployment.DeployResult, 100),
	}

	if err = daemons.Discovery(dctx); err != nil {
		return err
	}

	// this is a blocking operation until a certificate is acquired.
	if err = daemons.Autocert(dctx); err != nil {
		return err
	}

	if err = daemons.AgentCertificateCache(dctx); err != nil {
		return err
	}

	if err = daemons.Agent(dctx); err != nil {
		return err
	}

	if err = daemons.Bootstrap(dctx); err != nil {
		// if bootstrapping fails shutdown the process.
		return errors.Wrap(err, "failed to bootstrap node shutting down")
	}

	go deployment.ResultBus(dctx.Results, deployResults...)

	if err = daemons.Sync(dctx, cx); err != nil {
		// if syncing fails shutdown the process.
		return errors.Wrap(err, "failed to bootstrap node shutting down")
	}

	return nil
}


func (t *agentCmd) displayCmd(parent *kingpin.CmdClause) *kingpin.CmdClause {
	parent.Flag("verbose", "prints raft internal logs").Default("false").BoolVar(&t.raftVerbose)
	parent.Arg("path", "path to the raft log database").StringVar(&t.raftFile)
	return parent.Action(t.display)
}

func (t *agentCmd) display(ctx *kingpin.ParseContext) (err error) {
	type stats struct {
		barriers int
		commands int
		noops int
		configurations int
		unknown int
	}
	var (
		lstats stats
	)

	store, err := raftStoreFilepath(t.raftFile)
	if err != nil {
		return err
	}

	i, err := store.FirstIndex()
	if err != nil {
		return err
	}

	l, err := store.LastIndex()
	if err != nil {
		return err
	}

	for ; i <= l; i++ {
		var (
			current raft.Log
			decoded agent.Message
		)

		if err = store.GetLog(i, &current); err != nil {
			fmt.Println("get log failed", i, err)
			continue
		}

		switch current.Type {
		case raft.LogBarrier:
			lstats.barriers++
			if t.raftVerbose {
				fmt.Println("barrier invoked", current.Index, current.Term)
			}
			continue
		case raft.LogCommand:
			lstats.commands++
			if err = proto.Unmarshal(current.Data, &decoded); err != nil {
				fmt.Println("decode failed", i, err)
				continue
			}
			fmt.Println("message", proto.MarshalTextString(&decoded))
		case raft.LogNoop:
			lstats.noops++
			fmt.Println("noop invoked", current.Index, current.Term)
			continue
		case raft.LogConfiguration:
			lstats.configurations++
			if t.raftVerbose {
				fmt.Println("log configuration", current.Index, current.Term)
			}
		default:
			lstats.unknown++
			fmt.Println("unexpected log message", current.Type)
			continue
		}
	}

	fmt.Printf("log metrics %#v\n", lstats)
	return nil
}