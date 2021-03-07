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
	"github.com/james-lawrence/bw/internal/x/grpcx"
	"github.com/james-lawrence/bw/internal/x/logx"
	"github.com/james-lawrence/bw/internal/x/tlsx"
	"github.com/james-lawrence/bw/muxer"
	"github.com/james-lawrence/bw/notary"
	"github.com/james-lawrence/bw/storage"

	"github.com/alecthomas/kingpin"
	"github.com/davecgh/go-spew/spew"
	"github.com/golang/protobuf/proto"
	"github.com/hashicorp/raft"
	"github.com/pkg/errors"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials"
	"google.golang.org/grpc/keepalive"
)

type agentCmd struct {
	*global
	bootstrap   []*net.TCPAddr
	config      agent.Config
	configFile  string
	raftFile    string
	raftVerbose bool
}

func (t *agentCmd) configure(parent *kingpin.CmdClause) {
	t.global.cluster.configure(parent, &t.config)

	parent.Flag("agent-name", "name of the node within the network").Default(t.config.Name).StringVar(&t.config.Name)
	parent.Flag("agent-p2p", "address for the p2p server to bind to").PlaceHolder(
		t.config.P2PBind.String(),
	).Envar(
		bw.EnvAgentP2PBind,
	).TCPVar(&t.config.P2PBind)
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
	parent.Flag("agent-bootstrap", "addresses of the cluster to bootstrap from").PlaceHolder(
		t.config.P2PBind.String(),
	).Envar(
		"BEARDED_WOOKIE_P2P_BOOTSTRAP",
	).TCPListVar(&t.bootstrap)
	parent.Flag("agent-config", "file containing the agent configuration").
		Default(bw.DefaultLocation(filepath.Join(bw.DefaultEnvironmentName, bw.DefaultAgentConfig), "")).StringVar(&t.configFile)

	(&directive{agentCmd: t}).configure(parent.Command("directive", "directive based deployment").Default())

	t.displayCmd(parent.Command("quorum-state", "display the quorum state, only can be run on the server"))
	t.quorumCmd((parent.Command("quorum", "display quorum information, only can be run on the server")))
}

func (t *agentCmd) bind() (err error) {
	var (
		l        net.Listener
		p2ppriv  []byte
		p2ppub   []byte
		tc       storage.TorrentConfig
		tlscreds *tls.Config
		ns       notary.Composite
		ss       notary.Signer
		acmesvc  acme.DiskCache
	)

	if t.config, err = commandutils.LoadAgentConfig(t.configFile, t.config); err != nil {
		return err
	}

	log.SetPrefix("[AGENT] ")
	log.Println("configuration:", spew.Sdump(t.config))

	if err = bw.InitializeDeploymentDirectory(t.config.Root); err != nil {
		return err
	}

	// temporary certificate to allow bootstrapping a real certificate.
	if err = certificatecache.AutomaticTLSAgent(t.config.ServerName, t.config.CredentialsDir); err != nil {
		return err
	}

	if p2ppriv, err = rsax.CachedAuto(filepath.Join(t.config.Root, "p2p.pkey")); err != nil {
		return err
	}

	if p2ppub, err = sshx.PublicKey(p2ppriv); err != nil {
		return err
	}

	if ss, err = notary.NewSigner(p2ppriv); err != nil {
		return err
	}

	if ns, err = notary.NewFromFile(filepath.Join(t.config.Root, bw.DirAuthorizations), t.configFile); err != nil {
		return err
	}

	if _, err = ns.Insert(notary.AgentGrant(p2ppub)); err != nil {
		return err
	}

	if fingerprint, _, err := ss.AutoSignerInfo(); err != nil {
		return err
	} else {
		// important to maintain the agent name overtime and restarts.
		// otherwise raft will get upset over duplicate addresses for different.
		// server names.
		t.config = t.config.Clone(
			agent.ConfigOptionName(fingerprint),
		)
	}

	if tlscreds, err = daemons.TLSGenServer(t.config, tlsx.OptionNoClientCert, tlsx.OptionNextProtocols("bw.mux")); err != nil {
		return err
	}

	if acmesvc, err = acme.ReadConfig(t.config, t.configFile); err != nil {
		return err
	}

	local := cluster.NewLocal(
		agent.NewPeerFromTemplate(
			t.config.Peer(),
			agent.PeerOptionPublicKey(p2ppub),
		),
	)

	clusterevents := cluster.NewEventsQueue(local)

	if l, err = net.ListenTCP("tcp", t.config.P2PBind); err != nil {
		return err
	}

	// grpc can be insecure because the socket itself has tls.
	dialer := dialers.NewDefaults(
		dialers.WithMuxer(tlsx.NewDialer(tlscreds), l.Addr()),
		grpc.WithInsecure(),
		grpc.WithUnaryInterceptor(grpcx.DebugClientIntercepter),
		grpc.WithPerRPCCredentials(ss),
	)

	dctx := daemons.Context{
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
		DebugLog:          commandutils.DebugLog(t.global.debug),
		NotaryStorage:     ns,
		NotaryAuth:        notary.NewAuth(ns),
		RPCCredentials:    tlscreds,
		RPCKeepalive: keepalive.ServerParameters{
			MaxConnectionIdle: 1 * time.Hour,
			Time:              1 * time.Minute,
			Timeout:           2 * time.Minute,
		},
		Results:       make(chan deployment.DeployResult, 100),
		PeeringEvents: clusterevents,
		ACMECache:     acmesvc,
	}

	// go func(l net.Listener, err error) {
	// 	if err != nil {
	// 		panic(errors.Wrap(err, "default protocol registration failed"))
	// 	}
	// 	s := &http.Server{
	// 		Handler: http.HandlerFunc(func(resp http.ResponseWriter, req *http.Request) {
	// 			log.Println("HTTP REQUEST DETECTED")
	// 			http.NotFound(resp, req)
	// 		}),
	// 	}
	// 	s.Serve(l)
	// }(dctx.Muxer.Default("http", l.Addr()))

	if dctx, err = daemons.Inmem(dctx); err != nil {
		return errors.Wrap(err, "failed to initialize in memory services")
	}

	if dctx, err = daemons.Peering(dctx); err != nil {
		return errors.Wrap(err, "failed to initialize peering service")
	}

	// this unsafe dialer is used for the retrieving the response to an ACME challenge.
	// this is safe to do because:
	// 1. we'll only dial peers in our rendezous cluster. which have a pre shared key
	//    that validated them. (otherwise they couldn't be a member)
	// 2. the server we contact also ensures the client is a member of the cluster by validating the request signature.
	// 3. any individual agent only initiates challenges with servers that do have a valid TLS certificate.
	unsafedialer := dialers.NewDefaults(dialer.Defaults(dialers.WithMuxer(tlsx.NewDialer(tlscreds, tlsx.OptionInsecureSkipVerify), l.Addr()))...)
	l = tls.NewListener(
		l,
		certificatecache.NewALPN(
			tlscreds,
			acme.NewALPNCertCache(acme.NewResolver(t.config.Peer(), dctx.Cluster, acmesvc, unsafedialer)),
		),
	)

	dctx.MuxerListen(t.global.ctx, l)

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
	parent.Flag("verbose", "prints raft internal logs").Default("false").BoolVar(&t.raftVerbose)
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

	if creds, err = daemons.GRPCGenServer(t.config); err != nil {
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

func clearTorrents(c storage.TorrentConfig) chan deployment.DeployResult {
	var (
		tcu storage.TorrentUtil
		tdr = make(chan deployment.DeployResult)
	)

	go func() {
		for range tdr {
			tcu.ClearTorrents(c)
		}
	}()

	return tdr
}

func syncAuthorizations(ns notary.Composite) chan deployment.DeployResult {
	var (
		ndr = make(chan deployment.DeployResult)
	)

	go func() {
		for dr := range ndr {
			logx.MaybeLog(notary.CloneAuthorizationFile(filepath.Join(dr.Root, bw.DirArchive, bw.AuthKeysFile), filepath.Join(ns.Root, bw.AuthKeysFile)))
		}
	}()

	return ndr
}
