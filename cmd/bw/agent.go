package main

import (
	"crypto/tls"
	"fmt"
	"log"
	"net"
	"path/filepath"
	"time"

	"github.com/james-lawrence/bw"
	"github.com/james-lawrence/bw/agent"
	"github.com/james-lawrence/bw/agent/acme"
	"github.com/james-lawrence/bw/agent/dialers"
	"github.com/james-lawrence/bw/certificatecache"
	"github.com/james-lawrence/bw/cluster"
	"github.com/james-lawrence/bw/cmd/commandutils"
	"github.com/james-lawrence/bw/daemons"
	"github.com/james-lawrence/bw/deployment"
	"github.com/james-lawrence/bw/internal/rsax"
	"github.com/james-lawrence/bw/internal/sshx"
	"github.com/james-lawrence/bw/internal/x/envx"
	"github.com/james-lawrence/bw/internal/x/logx"
	"github.com/james-lawrence/bw/internal/x/tlsx"
	"github.com/james-lawrence/bw/muxer"
	"github.com/james-lawrence/bw/notary"
	"github.com/james-lawrence/bw/storage"

	"github.com/alecthomas/kingpin"
	"github.com/davecgh/go-spew/spew"
	"github.com/hashicorp/memberlist"
	"github.com/hashicorp/raft"
	"github.com/pkg/errors"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials"
	"google.golang.org/grpc/keepalive"
	"google.golang.org/protobuf/encoding/prototext"
	"google.golang.org/protobuf/proto"
)

type agentCmd struct {
	*global
	bootstrap      []*net.TCPAddr
	alternateBinds []*net.TCPAddr
	config         agent.Config
	configFile     string
	raftFile       string
}

func (t *agentCmd) configure(parent *kingpin.CmdClause) {
	t.global.cluster.configure(parent, &t.config)

	parent.Flag("agent-name", "name of the node within the network").Default(t.config.Name).StringVar(&t.config.Name)
	parent.Flag("agent-p2p", "address for the p2p server to bind to").PlaceHolder(
		t.config.P2PBind.String(),
	).Envar(
		bw.EnvAgentP2PBind,
	).TCPVar(&t.config.P2PBind)
	parent.Flag("agent-p2p-alternates", "alternate ip socket for the p2p server to bind to").PlaceHolder(
		"127.0.0.1:2000",
	).Envar(
		bw.EnvAgentP2PAlternatesBind,
	).TCPListVar(&t.alternateBinds)
	parent.Flag("agent-bootstrap", "addresses of the cluster to bootstrap from").PlaceHolder(
		t.config.P2PBind.String(),
	).Envar(
		"BEARDED_WOOKIE_P2P_BOOTSTRAP",
	).TCPListVar(&t.bootstrap)
	parent.Flag("agent-config", "file containing the agent configuration").
		Default(bw.DefaultLocation(filepath.Join(bw.DefaultEnvironmentName, bw.DefaultAgentConfig), "")).StringVar(&t.configFile)

	(&agentDeploymentRuntime{agentCmd: t}).configure(parent.Command("directive", "directive based deployment").Hidden())
	(&agentDeploymentRuntime{agentCmd: t}).configure(parent.Command("deploy", "run the deploy agent").Default())
	(&agentDeploymentCache{agentCmd: t}).configure(parent.Command("command", "run a server that purely acts as a command and control node, you deploy to the cluster and it'll store the archive but not actually execute it.").Default())

	t.displayCmd(parent.Command("quorum-state", "display the quorum state, only can be run on the server"))
	t.quorumCmd((parent.Command("quorum", "display quorum information, only can be run on the server")))
}

func (t *agentCmd) bind(deployer daemons.Deployer) (err error) {
	var (
		ring      *memberlist.Keyring
		l         net.Listener
		bound     []net.Listener
		localpriv []byte
		localpub  []byte
		tc        storage.TorrentConfig
		tlscreds  *tls.Config
		ns        notary.Composite
		ss        notary.Signer
		acmesvc   acme.DiskCache
	)

	if t.config, err = commandutils.LoadAgentConfig(t.configFile, t.config); err != nil {
		return err
	}

	log.SetPrefix("[AGENT] ")
	log.Println("configuration:", spew.Sdump(t.config.Sanitize()))

	if err = bw.InitializeDeploymentDirectory(t.config.Root); err != nil {
		return err
	}

	if ring, err = t.config.Keyring(); err != nil {
		return err
	}

	// temporary certificate to allow bootstrapping a real certificate.
	if err = certificatecache.AutomaticTLSAgent(ring.GetPrimaryKey(), t.config.ServerName, t.config.CredentialsDir); err != nil {
		return err
	}

	if localpriv, err = rsax.CachedAuto(filepath.Join(t.config.Root, bw.DefaultAgentNotaryKey)); err != nil {
		return err
	}

	if localpub, err = sshx.PublicKey(localpriv); err != nil {
		return err
	}

	// important to maintain the agent name overtime and restarts.
	// otherwise raft will get upset over duplicate addresses for different.
	// server names.
	t.config = t.config.Clone(
		agent.ConfigOptionName(sshx.FingerprintSHA256(localpub)),
	)

	if ns, err = notary.NewFromFile(filepath.Join(t.config.Root, bw.DirAuthorizations), t.configFile); err != nil {
		return err
	}

	if ss, err = generatecredentials(t.config, ns); err != nil {
		return err
	}

	if tlscreds, err = certificatecache.TLSGenServer(t.config, tlsx.OptionNoClientCert); err != nil {
		return err
	}

	if acmesvc, err = acme.ReadConfig(t.config, t.configFile); err != nil {
		return err
	}

	local := cluster.NewLocal(
		t.config.Peer(),
	)

	clusterevents := cluster.NewEventsQueue(local)

	if l, err = net.ListenTCP("tcp", t.config.P2PBind); err != nil {
		return err
	}
	bound = append(bound, l)

	log.Println("alternate bindings", len(t.alternateBinds))
	for _, alt := range t.alternateBinds {
		var (
			l2 net.Listener
		)

		if l2, err = net.ListenTCP("tcp", alt); err != nil {
			return err
		}

		bound = append(bound, l2)
	}

	// grpc can be insecure because the socket itself has tls.
	dialer := dialers.NewDefaults(
		dialers.WithMuxer(tlsx.NewDialer(tlscreds), l.Addr()),
		grpc.WithInsecure(),
		grpc.WithPerRPCCredentials(ss),
	)

	dctx := daemons.Context{
		Deploys:           deployer,
		Local:             local,
		Listener:          l,
		Dialer:            dialer,
		Muxer:             muxer.New(),
		ConfigurationFile: t.configFile,
		Config:            t.config,
		Context:           t.global.ctx,
		Shutdown:          t.global.shutdown,
		Cleanup:           t.global.cleanup,
		Debug:             t.global.debug,
		DebugLog:          commandutils.DebugLog(envx.Boolean(t.global.debug, bw.EnvLogsGossip)),
		NotaryStorage:     ns,
		NotaryAuth:        notary.NewAuth(ns),
		RPCCredentials:    tlscreds,
		RPCKeepalivePolicy: keepalive.EnforcementPolicy{
			MinTime:             10 * time.Second,
			PermitWithoutStream: true,
		},
		RPCKeepalive: keepalive.ServerParameters{
			MaxConnectionIdle: 1 * time.Hour,
			Time:              1 * time.Minute,
		},
		Results:       make(chan *deployment.DeployResult, 100),
		PeeringEvents: clusterevents,
		ACMECache:     acmesvc,
	}

	if dctx, err = daemons.Proxy(dctx, tlsx.NewDialer(tlscreds)); err != nil {
		return errors.Wrap(err, "failed to initialize proxy connection service")
	}

	if dctx, err = daemons.Inmem(dctx); err != nil {
		return errors.Wrap(err, "failed to initialize in memory services")
	}

	if dctx, err = daemons.Peering(dctx); err != nil {
		return errors.Wrap(err, "failed to initialize peering service")
	}

	alpn := certificatecache.NewALPN(
		tlscreds,
		acme.NewALPNCertCache(acme.NewResolver(t.config.Peer(), dctx.Cluster, acmesvc, dialer)),
	)

	for idx, b := range bound {
		bound[idx] = tls.NewListener(
			b,
			alpn,
		)
	}

	dctx.MuxerListen(t.global.ctx, bound...)

	if dctx, err = daemons.Peered(dctx, t.global.cluster); err != nil {
		return errors.Wrap(err, "failed to initialize peering service")
	}

	if dctx, err = daemons.Quorum(dctx, t.global.cluster); err != nil {
		return errors.Wrap(err, "failed to initialize quorum service")
	}

	if err = daemons.Discovery(dctx); err != nil {
		return errors.Wrap(err, "failed to initialize discovery service")
	}

	if err = daemons.Autocert(dctx); err != nil {
		return errors.Wrap(err, "failed to initialize autocert service")
	}

	// this is a blocking operation until a certificate is acquired.
	if err = daemons.AgentCertificateCache(dctx); err != nil {
		return errors.Wrap(err, "failed to initialize certificate cache service")
	}

	if tc, err = daemons.Torrent(dctx); err != nil {
		return errors.Wrap(err, "failed to initialize deploy archive transfer service")
	}

	if err = daemons.Agent(dctx, tc.Uploader(), tc.Downloader()); err != nil {
		return errors.Wrap(err, "failed to initialize agent service")
	}

	go deployment.ResultBus(
		dctx.Results,
		syncAuthorizations(ns),
		clearTorrents(tc),
	)

	if err = daemons.Bootstrap(dctx, tc.Downloader()); err != nil {
		return errors.Wrap(err, "failed to bootstrap node shutting down")
	}

	return nil
}

func (t *agentCmd) displayCmd(parent *kingpin.CmdClause) *kingpin.CmdClause {
	parent.Arg("path", "path to the raft log database").StringVar(&t.raftFile)
	return parent.Action(t.display)
}

func (t *agentCmd) display(ctx *kingpin.ParseContext) (err error) {
	type stats struct {
		barriers       int
		commands       int
		noops          int
		configurations int
		unknown        int
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
			if envx.Boolean(false, bw.EnvLogsVerbose) {
				fmt.Println("barrier invoked", current.Index, current.Term)
			}
			continue
		case raft.LogCommand:
			lstats.commands++
			if err = proto.Unmarshal(current.Data, &decoded); err != nil {
				fmt.Println("decode failed", i, err)
				continue
			}
			fmt.Println("message", prototext.Format(&decoded))
		case raft.LogNoop:
			lstats.noops++
			fmt.Println("noop invoked", current.Index, current.Term)
			continue
		case raft.LogConfiguration:
			lstats.configurations++
			if envx.Boolean(false, bw.EnvLogsVerbose) {
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

func (t *agentCmd) quorumCmd(parent *kingpin.CmdClause) *kingpin.CmdClause {
	return parent.Action(t.quorum)
}

func (t *agentCmd) quorum(ctx *kingpin.ParseContext) (err error) {
	var (
		conn   *grpc.ClientConn
		d      dialers.Dialer
		creds  credentials.TransportCredentials
		quorum *agent.InfoResponse
	)
	defer t.global.shutdown()

	if t.config, err = commandutils.LoadAgentConfig(t.configFile, t.config); err != nil {
		return errors.Wrap(err, "unable to load configuration")
	}

	log.Println(spew.Sdump(t.config))

	if creds, err = certificatecache.GRPCGenServer(t.config); err != nil {
		return err
	}

	d = dialers.NewDirect(
		agent.RPCAddress(t.config.Peer()),
		grpc.WithTransportCredentials(creds),
	)

	if conn, err = d.Dial(); err != nil {
		return err
	}

	if quorum, err = agent.NewQuorumClient(conn).Info(t.global.ctx, &agent.InfoRequest{}); err != nil {
		return err
	}

	fmt.Println("quorum:")
	for idx, p := range quorum.Quorum {
		log.Println(idx, p.Name, spew.Sdump(p))
	}

	peer := func(p *agent.Peer) string {
		if p == nil {
			return "None"
		}

		return fmt.Sprintf("peer %s - %s", p.Name, spew.Sdump(p))
	}

	deployment := func(c *agent.DeployCommand) string {
		if c == nil || c.Archive == nil {
			return "None"
		}

		return fmt.Sprintf("deployment %s - %s - %s", bw.RandomID(c.Archive.DeploymentID), c.Archive.Initiator, c.Command.String())
	}

	fmt.Printf("leader: %s\n", peer(quorum.Leader))
	fmt.Printf("latest: %s\n", deployment(quorum.Deployed))
	fmt.Printf("ongoing: %s\n", deployment(quorum.Deploying))

	return nil
}

func clearTorrents(c storage.TorrentConfig) chan *deployment.DeployResult {
	var (
		tcu storage.TorrentUtil
		tdr = make(chan *deployment.DeployResult)
	)

	go func() {
		for range tdr {
			tcu.ClearTorrents(c)
		}
	}()

	return tdr
}

func syncAuthorizations(ns notary.Composite) chan *deployment.DeployResult {
	var (
		ndr = make(chan *deployment.DeployResult)
	)

	go func() {
		for dr := range ndr {
			logx.MaybeLog(notary.CloneAuthorizationFile(filepath.Join(dr.Root, bw.DirArchive, bw.AuthKeysFile), filepath.Join(ns.Root, bw.AuthKeysFile)))
		}
	}()

	return ndr
}
