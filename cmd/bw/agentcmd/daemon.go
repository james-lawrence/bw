package agentcmd

import (
	"context"
	"crypto/tls"
	"log"
	"net"
	"path/filepath"
	"time"

	"github.com/bits-and-blooms/bloom/v3"
	"github.com/james-lawrence/bw"
	"github.com/james-lawrence/bw/agent"
	"github.com/james-lawrence/bw/agent/acme"
	"github.com/james-lawrence/bw/agent/dialers"
	"github.com/james-lawrence/bw/certificatecache"
	"github.com/james-lawrence/bw/cluster"
	"github.com/james-lawrence/bw/cmd/bw/cmdopts"
	"github.com/james-lawrence/bw/cmd/commandutils"
	"github.com/james-lawrence/bw/daemons"
	"github.com/james-lawrence/bw/deployment"
	"github.com/james-lawrence/bw/directives/shell"
	"github.com/james-lawrence/bw/internal/envx"
	"github.com/james-lawrence/bw/internal/errorsx"
	"github.com/james-lawrence/bw/internal/rsax"
	"github.com/james-lawrence/bw/internal/sshx"
	"github.com/james-lawrence/bw/internal/tlsx"
	"github.com/james-lawrence/bw/muxer"
	"github.com/james-lawrence/bw/notary"
	"github.com/james-lawrence/bw/storage"

	"github.com/davecgh/go-spew/spew"
	"github.com/hashicorp/memberlist"
	"github.com/pkg/errors"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/keepalive"
)

type CmdDaemon struct {
	Runtime     CmdRuntime           `cmd:"" help:"run the deploy agent runtime" default:"true" aliases:"deploy"`
	Coordinator CmdCoordinator       `cmd:"" help:"run a coordination server that purely acts as a command and control node, a deploy to the cluster will store the archive but not actually process it within the deployment runtime"`
	QuorumLog   CmdDaemonDebugRaft   `cmd:"" name:"quorum-state" help:"display the quorum log, only runs on the server"`
	Quorum      CmdDaemonDebugQuorum `cmd:"" name:"quorum" help:"display quorum member information, only runs on the server"`
}

type CmdRuntime struct {
	daemon
}

func (t CmdRuntime) Run(ctx *cmdopts.Global, aconfig *agent.Config) (err error) {
	var (
		sctx shell.Context
	)

	if sctx, err = shell.DefaultContext(); err != nil {
		return err
	}

	deployer := deployment.NewDirective(
		deployment.DirectiveOptionShellContext(sctx),
	)

	return t.daemon.bind(ctx, aconfig.Clone(), deployer)
}

type CmdCoordinator struct {
	daemon
}

func (t CmdCoordinator) Run(ctx *cmdopts.Global, aconfig *agent.Config) (err error) {
	return t.daemon.bind(ctx, aconfig.Clone(), deployment.Cached{})
}

type daemon struct {
	cmdopts.Peering
	Config
}

func (t *daemon) bind(ctx *cmdopts.Global, config agent.Config, deployer daemons.Deployer) (err error) {
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

	if config, err = commandutils.LoadAgentConfig(t.Location, config); err != nil {
		return err
	}

	log.SetPrefix("[AGENT] ")
	log.Println("configuration:", spew.Sdump(config.Sanitize()))

	if err = bw.InitializeDeploymentDirectory(config.Root); err != nil {
		return err
	}

	if ring, err = config.Keyring(); err != nil {
		return err
	}

	// temporary certificate to allow bootstrapping a real certificate.
	if err = certificatecache.AutomaticTLSAgent(ring.GetPrimaryKey(), config.ServerName, config.CredentialsDir); err != nil {
		return err
	}

	if localpriv, err = rsax.CachedAuto(filepath.Join(config.Root, bw.DefaultAgentNotaryKey)); err != nil {
		return err
	}

	if localpub, err = sshx.PublicKey(localpriv); err != nil {
		return err
	}

	// important to maintain the agent name overtime and restarts.
	// otherwise raft will get upset over duplicate addresses for different.
	// server names.
	config = config.Clone(
		agent.ConfigOptionName(sshx.FingerprintSHA256(localpub)),
	)

	if ns, err = notary.NewFromFile(filepath.Join(config.Root, bw.DirAuthorizations), t.Location); err != nil {
		return err
	}

	if ss, err = commandutils.Generatecredentials(config, ns); err != nil {
		return err
	}

	if tlscreds, err = certificatecache.TLSGenServer(config, tlsx.OptionNoClientCert); err != nil {
		return err
	}

	local := cluster.NewLocal(
		config.Peer(),
	)

	clusterevents := cluster.NewEventsQueue(local)

	if l, err = net.ListenTCP("tcp", config.P2PBind); err != nil {
		return err
	}
	bound = append(bound, l)

	log.Println("alternate bindings", len(config.AlternateBinds))
	for _, alt := range config.AlternateBinds {
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
		grpc.WithTransportCredentials(insecure.NewCredentials()),
		grpc.WithPerRPCCredentials(ss),
	)

	if acmesvc, err = acme.ReadConfig(config, t.Location); err != nil {
		return err
	}

	dctx := daemons.Context{
		Deploys:           deployer,
		Local:             local,
		Listener:          l,
		Dialer:            dialer,
		Muxer:             muxer.New(),
		ConfigurationFile: t.Location,
		Config:            config,
		Context:           ctx.Context,
		Shutdown:          ctx.Shutdown,
		Cleanup:           ctx.Cleanup,
		DebugLog:          commandutils.DebugLog(envx.Boolean(false, bw.EnvLogsGossip)),
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
		acme.NewALPNCertCache(acme.NewResolver(config.Peer(), dctx.Cluster, acmesvc, dialer)),
	)

	for idx, b := range bound {
		bound[idx] = tls.NewListener(
			b,
			alpn,
		)
	}

	dctx.MuxerListen(ctx.Context, bound...)

	if err = daemons.Discovery(dctx); err != nil {
		return errors.Wrap(err, "failed to initialize discovery service")
	}

	if dctx, err = daemons.Peered(dctx, &t.Peering); err != nil {
		return errors.Wrap(err, "failed to initialize peering service")
	}

	if dctx, err = daemons.Quorum(dctx, &t.Peering); err != nil {
		return errors.Wrap(err, "failed to initialize quorum service")
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
		syncAuthorizations(dctx, ns),
		clearTorrents(tc),
	)

	if err = daemons.Bootstrap(dctx, tc.Downloader()); err != nil {
		return errors.Wrap(err, "failed to bootstrap node shutting down")
	}

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

func syncAuthorizations(dctx daemons.Context, ns notary.Composite) chan *deployment.DeployResult {
	var (
		ndr = make(chan *deployment.DeployResult)
		// sync credentials between servers
		b = bloom.NewWithEstimates(1000, 0.0001)
	)

	go func() {
		for dr := range ndr {
			errorsx.MaybeLog(notary.CloneAuthorizationFile(filepath.Join(dr.Root, bw.DirArchive, bw.AuthKeysFile), filepath.Join(ns.Root, bw.AuthKeysFile)))

			for _, p := range agent.RendezvousPeers(dctx.P2PPublicKey, dctx.Cluster) {
				// Notary Subscriptions to node events. syncs authorization between agents
				req, err := notary.NewSyncRequest(b)
				if err != nil {
					log.Println("unable generate request", err)
					continue
				}

				d := dialers.NewDirect(agent.RPCAddress(p), dctx.Dialer.Defaults()...)
				ctx, done := context.WithTimeout(context.Background(), 5*time.Minute)
				conn, err := d.DialContext(ctx)
				done()
				if err != nil {
					log.Println("unable to connect", err)
					continue
				}

				client := notary.NewSyncClient(conn)
				stream, err := client.Stream(context.Background(), req)
				if err != nil {
					log.Println("unable to stream", err)
					continue
				}

				log.Println("syncing credentials initiated", agent.RPCAddress(p))
				err = notary.Sync(stream, b, dctx.NotaryStorage)
				if err != nil {
					log.Println("syncing credentials failed", err)
					continue
				} else {
					log.Println("syncing credentials completed", agent.RPCAddress(p))
				}
			}
		}
	}()

	return ndr
}
