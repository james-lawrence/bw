package daemons

import (
	"context"
	"log"
	"net"
	"path/filepath"
	"time"

	"github.com/hashicorp/raft"
	"github.com/pkg/errors"
	"google.golang.org/grpc"

	"github.com/james-lawrence/bw"
	"github.com/james-lawrence/bw/agent"
	"github.com/james-lawrence/bw/agent/acme"
	"github.com/james-lawrence/bw/agent/debug"
	"github.com/james-lawrence/bw/agent/dialers"
	"github.com/james-lawrence/bw/agent/journal"
	"github.com/james-lawrence/bw/agent/proxy"
	"github.com/james-lawrence/bw/agent/quorum"
	"github.com/james-lawrence/bw/agentutil"
	"github.com/james-lawrence/bw/certificatecache"
	"github.com/james-lawrence/bw/clustering"
	"github.com/james-lawrence/bw/deployment"
	"github.com/james-lawrence/bw/internal/errorsx"
	"github.com/james-lawrence/bw/notary"
	"github.com/james-lawrence/bw/storage"
)

// Agent daemon - rpc endpoint for the system.
func Agent(dctx Context, upload storage.UploadProtocol, download storage.DownloadProtocol) (err error) {
	var (
		bind     net.Listener
		journald *journal.Journal
		dlreg    = storage.New(storage.OptionProtocols(download))
	)

	if journald, err = journal.New(filepath.Join(dctx.Config.RuntimeDirectory, "journal.d")); err != nil {
		return err
	}

	// the journal serves for the lifetime of the daemon, Agent returns as soon as the servers are bound
	// so it can't be closed on return. close it with the rest of the daemon.
	dctx.Cleanup.Add(1)
	context.AfterFunc(dctx.Context, func() {
		defer dctx.Cleanup.Done()
		errorsx.Log(errors.Wrap(journald.Close(), "journal shutdown failed"))
	})

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
		journald,
		dctx.Cluster,
		quorum.NewTranscoder(
			quorum.Logging{},
		),
		upload,
		dctx.Raft,
		quorum.OptionDialer(qdialer),
		quorum.OptionModeChange(func(m agent.PeerMode) {
			if err := dctx.Local.SetMode(m); err != nil {
				log.Println("unable to update node mode", err)
				return
			}

			// informational, so advertising is best effort and happens in the background.
			if r, ok := dctx.Bootstrapper.(clustering.Replication); ok {
				go func() {
					if err := r.UpdateNode(5 * time.Second); err != nil {
						log.Println("unable to advertise node mode", err)
					}
				}()
			}
		}),
	)
	go (&q).Observe(make(chan raft.Observation, 200))

	agent.NewQuorum(
		&q,
		notary.NewAgentAuth(dctx.NotaryAuth),
	).Bind(server)

	notary.New(
		dctx.Config.ServerName,
		certificatecache.NewAuthorityCache(dctx.Ring.GetPrimaryKey(), dctx.Config.ServerName, dctx.Config.Credentials.Directory),
		dctx.NotaryStorage,
	).Bind(server)

	notary.NewSyncService(
		dctx.NotaryAuth,
		dctx.NotaryStorage,
	).Bind(server)

	debug.NewService(
		notary.NewAgentAuth(dctx.NotaryAuth),
	).Bind(server)

	proxy.NewDeployment(dctx.NotaryAuth, qdialer).Bind(server)
	acme.NewService(dctx.ACMECache, dctx.NotaryAuth).Bind(server)

	if bind, err = dctx.Muxer.Bind(bw.ProtocolAgent, dctx.Listener.Addr()); err != nil {
		return errors.Wrap(err, "failed to bind agent protocol")
	}

	dctx.grpc("agent", server, bind)

	return nil
}
