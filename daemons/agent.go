package daemons

import (
	"log"
	"net"
	"path/filepath"
	"time"

	"github.com/hashicorp/raft"
	"github.com/pkg/errors"
	"google.golang.org/grpc"

	"github.com/james-lawrence/bw/agent"
	"github.com/james-lawrence/bw/agent/observers"
	"github.com/james-lawrence/bw/agent/proxy"
	"github.com/james-lawrence/bw/agent/quorum"
	"github.com/james-lawrence/bw/agentutil"
	"github.com/james-lawrence/bw/deployment"
	"github.com/james-lawrence/bw/directives/shell"
	"github.com/james-lawrence/bw/internal/x/logx"
	"github.com/james-lawrence/bw/internal/x/timex"
	"github.com/james-lawrence/bw/storage"
)

// Agent daemon - rpc endpoint for the system.
func Agent(ctx Context, config agent.Config) (err error) {
	var (
		sctx         shell.Context
		observersdir observers.Directory
		bind         net.Listener
		dlreg        = storage.New(storage.OptionProtocols(ctx.Download))
	)

	if sctx, err = shell.DefaultContext(); err != nil {
		return err
	}

	if observersdir, err = observers.NewDirectory(filepath.Join(config.Root, "observers")); err != nil {
		return err
	}

	keepalive := grpc.KeepaliveParams(ctx.RPCKeepalive)

	dialer := agent.NewDialer(agent.DefaultDialerOptions(grpc.WithTransportCredentials(ctx.RPCCredentials))...)
	qdialer := agent.NewQuorumDialer(dialer)
	dispatcher := agentutil.NewDispatcher(ctx.Cluster, qdialer)

	deploy := deployment.NewDirective(
		deployment.DirectiveOptionShellContext(sctx),
	)

	coordinator := deployment.New(
		config.Peer(),
		deploy,
		deployment.CoordinatorOptionDispatcher(dispatcher),
		deployment.CoordinatorOptionRoot(config.Root),
		deployment.CoordinatorOptionKeepN(config.KeepN),
		deployment.CoordinatorOptionDeployResults(ctx.Results),
		deployment.CoordinatorOptionStorage(dlreg),
	)

	authority := quorum.NewAuthority(config)

	configuration := quorum.NewConfiguration(authority, ctx.Cluster, dialer)
	configurationsvc := quorum.NewConfigurationService(configuration)

	q := quorum.New(
		observersdir,
		ctx.Cluster,
		proxy.NewProxy(ctx.Cluster),
		quorum.NewTranscoder(
			authority,
			configuration,
		),
		ctx.Upload,
		quorum.OptionDialer(dialer),
		quorum.OptionInitializers(
			authority,
		),
	)
	go (&q).Observe(ctx.Raft, make(chan raft.Observation, 200))

	a := agent.NewServer(
		ctx.Cluster,
		agent.ServerOptionDeployer(&coordinator),
		agent.ServerOptionShutdown(ctx.Shutdown),
	)

	aq := agent.NewQuorum(&q)

	server := grpc.NewServer(grpc.Creds(ctx.RPCCredentials), keepalive)
	agent.RegisterAgentServer(server, a)
	agent.RegisterQuorumServer(server, aq)
	agent.RegisterConfigurationServer(server, configurationsvc)

	if bind, err = net.Listen(config.RPCBind.Network(), config.RPCBind.String()); err != nil {
		return errors.Wrapf(err, "failed to bind agent to %s", config.RPCBind)
	}

	// hack to propagate TLS to agents who are not in the quorum.
	// this can be removed once RPC credentials is fully implemented.
	go timex.NowAndEvery(1*time.Hour, func() {
		if agent.DetectQuorum(ctx.Cluster, agent.IsInQuorum(ctx.Cluster.Local())) != nil {
			log.Println("tls request skipped")
			return
		}

		log.Println("tls request initiated")
		defer log.Println("tls request completed")

		logx.MaybeLog(
			errors.Wrap(
				agentutil.ReliableDispatch(
					ctx.Context, dispatcher,
					agentutil.TLSRequest(ctx.Cluster.Local()),
				),
				"failed to dispatch tls request",
			),
		)
	})

	ctx.grpc("agent", server, bind)

	return nil
}
