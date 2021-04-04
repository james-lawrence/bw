package daemons

import (
	"context"
	"net"
	"path/filepath"
	"time"

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
	"github.com/james-lawrence/bw/certificatecache"
	"github.com/james-lawrence/bw/deployment"
	"github.com/james-lawrence/bw/directives/shell"
	"github.com/james-lawrence/bw/internal/x/grpcx"
	"github.com/james-lawrence/bw/internal/x/logx"
	"github.com/james-lawrence/bw/internal/x/timex"
	"github.com/james-lawrence/bw/notary"
	"github.com/james-lawrence/bw/storage"
)

// Agent daemon - rpc endpoint for the system.
func Agent(ctx Context, upload storage.UploadProtocol, download storage.DownloadProtocol) (err error) {
	var (
		bind         net.Listener
		sctx         shell.Context
		observersdir observers.Directory
		dlreg        = storage.New(storage.OptionProtocols(download))
	)

	if sctx, err = shell.DefaultContext(); err != nil {
		return err
	}

	if observersdir, err = observers.NewDirectory(filepath.Join(ctx.Config.Root, "observers")); err != nil {
		return err
	}

	qdialer := dialers.NewQuorum(
		ctx.Cluster,
		ctx.Dialer.Defaults()...,
	)
	dialer := agent.NewDialer(ctx.Dialer.Defaults()...)
	dispatcher := agentutil.NewDispatcher(qdialer)

	deploy := deployment.NewDirective(
		deployment.DirectiveOptionShellContext(sctx),
	)

	coordinator := deployment.New(
		ctx.Config.Peer(),
		deploy,
		deployment.CoordinatorOptionDispatcher(dispatcher),
		deployment.CoordinatorOptionRoot(ctx.Config.Root),
		deployment.CoordinatorOptionKeepN(ctx.Config.KeepN),
		deployment.CoordinatorOptionDeployResults(ctx.Results),
		deployment.CoordinatorOptionStorage(dlreg),
	)

	server := grpc.NewServer(
		grpc.UnaryInterceptor(grpcx.DebugIntercepter),
		grpc.StreamInterceptor(grpcx.DebugStreamIntercepter),
		grpc.KeepaliveParams(ctx.RPCKeepalive),
	)
	authority := quorum.NewAuthority(ctx.Config)

	configuration := quorum.NewConfiguration(authority, ctx.Cluster, dialer)
	configurationsvc := quorum.NewConfigurationService(configuration)

	agent.NewServer(
		ctx.Cluster,
		agent.ServerOptionAuth(notary.NewAgentAuth(ctx.NotaryAuth)),
		agent.ServerOptionDeployer(&coordinator),
		agent.ServerOptionShutdown(ctx.Shutdown),
	).Bind(server)

	q := quorum.New(
		observersdir,
		ctx.Cluster,
		proxy.NewProxy(ctx.Cluster),
		quorum.NewTranscoder(
			authority,
			configuration,
		),
		upload,
		ctx.Raft,
		quorum.OptionDialer(qdialer),
		quorum.OptionInitializers(
			authority,
		),
	)
	go (&q).Observe(make(chan raft.Observation, 200))

	agent.NewQuorum(
		&q,
		notary.NewAgentAuth(ctx.NotaryAuth),
	).Bind(server)

	notary.New(
		ctx.Config.ServerName,
		certificatecache.NewAuthorityCache(ctx.Config.CredentialsDir),
		ctx.NotaryStorage,
	).Bind(server)

	proxy.NewDeployment(ctx.NotaryAuth, qdialer).Bind(server)

	agent.RegisterConfigurationServer(server, configurationsvc)
	acme.RegisterACMEServer(server, acme.NewService(ctx.ACMECache, ctx.NotaryAuth))

	// hack to propagate TLS to agents who are not in the quorum.
	// this can be removed once per request credentials is fully implemented.
	go timex.NowAndEvery(1*time.Hour, func() {
		attempt := func() error {
			// request the TLS certificate from the cluster.
			// log.Println("tls request initiated")
			// defer log.Println("tls request completed")

			tctx, done := context.WithTimeout(ctx.Context, 10*time.Second)
			defer done()
			return errors.Wrap(
				agentutil.ReliableDispatch(
					tctx, dispatcher,
					agentutil.TLSRequest(ctx.Cluster.Local()),
				),
				"failed to dispatch tls request",
			)
		}

		for deadline := time.Now().Add(10 * time.Minute); deadline.After(time.Now()); {
			if agent.DetectQuorum(ctx.Cluster, agent.IsInQuorum(ctx.Cluster.Local())) != nil {
				// skip tls request
				return
			}

			if logx.MaybeLog(attempt()) == nil {
				return
			}
		}
	})

	if bind, err = ctx.Muxer.Bind(bw.ProtocolAgent, ctx.Listener.Addr()); err != nil {
		return errors.Wrap(err, "failed to bind agent protocol")
	}

	ctx.grpc("agent", server, bind)

	return nil
}
