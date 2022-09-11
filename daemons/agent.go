package daemons

import (
	"context"
	"log"
	"net"
	"time"

	"github.com/bits-and-blooms/bloom/v3"
	"github.com/hashicorp/raft"
	"github.com/pkg/errors"
	"google.golang.org/grpc"

	"github.com/james-lawrence/bw"
	"github.com/james-lawrence/bw/agent"
	"github.com/james-lawrence/bw/agent/acme"
	"github.com/james-lawrence/bw/agent/dialers"
	"github.com/james-lawrence/bw/agent/observers"
	"github.com/james-lawrence/bw/agent/proxy"
	"github.com/james-lawrence/bw/agent/quorum"
	"github.com/james-lawrence/bw/agentutil"
	"github.com/james-lawrence/bw/backoff"
	"github.com/james-lawrence/bw/certificatecache"
	"github.com/james-lawrence/bw/deployment"
	"github.com/james-lawrence/bw/notary"
	"github.com/james-lawrence/bw/storage"
)

// Agent daemon - rpc endpoint for the system.
func Agent(dctx Context, upload storage.UploadProtocol, download storage.DownloadProtocol) (err error) {
	var (
		bind         net.Listener
		observersmem observers.Memory
		dlreg        = storage.New(storage.OptionProtocols(download))
	)

	if observersmem, err = observers.NewMemory(); err != nil {
		return err
	}

	qdialer := dialers.NewQuorum(
		dctx.Cluster,
		dctx.Dialer.Defaults()...,
	)
	dispatcher := agentutil.NewDispatcher(qdialer)

	coordinator := deployment.New(
		dctx.Config.Peer(),
		dctx.Deploys,
		deployment.CoordinatorOptionDispatcher(dispatcher),
		deployment.CoordinatorOptionRoot(dctx.Config.Root),
		deployment.CoordinatorOptionKeepN(dctx.Config.KeepN),
		deployment.CoordinatorOptionDeployResults(dctx.Results),
		deployment.CoordinatorOptionStorage(dlreg),
	)

	server := grpc.NewServer(
		// grpc.UnaryInterceptor(grpcx.DebugIntercepter),
		// grpc.StreamInterceptor(grpcx.DebugStreamIntercepter),
		grpc.KeepaliveParams(dctx.RPCKeepalive),
		grpc.KeepaliveEnforcementPolicy(dctx.RPCKeepalivePolicy),
	)

	agent.NewServer(
		dctx.Cluster,
		agent.ServerOptionAuth(notary.NewAgentAuth(dctx.NotaryAuth)),
		agent.ServerOptionDeployer(&coordinator),
		agent.ServerOptionShutdown(dctx.Shutdown),
	).Bind(server)

	q := quorum.New(
		observersmem,
		dctx.Cluster,
		proxy.NewProxy(dctx.Cluster),
		quorum.NewTranscoder(
			quorum.Logging{},
		),
		upload,
		dctx.Raft,
		quorum.OptionDialer(qdialer),
	)
	go (&q).Observe(make(chan raft.Observation, 200))

	agent.NewQuorum(
		&q,
		notary.NewAgentAuth(dctx.NotaryAuth),
	).Bind(server)

	notary.New(
		dctx.Config.ServerName,
		certificatecache.NewAuthorityCache(dctx.Config.Name, dctx.Config.CredentialsDir),
		dctx.NotaryStorage,
	).Bind(server)

	notary.NewSyncService(
		dctx.NotaryAuth,
		dctx.NotaryStorage,
	).Bind(server)

	proxy.NewDeployment(dctx.NotaryAuth, qdialer).Bind(server)
	acme.NewService(dctx.ACMECache, dctx.NotaryAuth).Bind(server)

	// sync credentials between servers
	b := bloom.NewWithEstimates(1000, 0.0001)
	s := backoff.New(
		backoff.Exponential(time.Minute),
		backoff.Maximum(8*time.Hour),
		backoff.Jitter(0.25),
	)

	go backoff.Attempt(s, func(previous int) int {
		for _, p := range agent.SynchronizationPeers(dctx.P2PPublicKey, dctx.Cluster) {
			// Notary Subscriptions to node events. syncs authorization between agents
			req, err := notary.NewSyncRequest(b)
			if err != nil {
				log.Println("unable generate request", err)
				return 0
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

		return previous + 1
	})

	if bind, err = dctx.Muxer.Bind(bw.ProtocolAgent, dctx.Listener.Addr()); err != nil {
		return errors.Wrap(err, "failed to bind agent protocol")
	}

	dctx.grpc("agent", server, bind)

	return nil
}
