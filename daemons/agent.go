package daemons

import (
	"net"
	"path/filepath"

	"github.com/james-lawrence/bw/agent"
	"github.com/james-lawrence/bw/agent/observers"
	"github.com/james-lawrence/bw/agent/proxy"
	"github.com/james-lawrence/bw/agent/quorum"
	"github.com/james-lawrence/bw/agentutil"
	"github.com/james-lawrence/bw/deployment"
	"github.com/james-lawrence/bw/directives/shell"
	"github.com/james-lawrence/bw/storage"
	"github.com/pkg/errors"

	"github.com/hashicorp/raft"
	"google.golang.org/grpc"
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
	q := quorum.New(
		observersdir,
		ctx.Cluster,
		proxy.NewProxy(ctx.Cluster),
		quorum.NewTranscoder(
			authority,
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

	if bind, err = net.Listen(config.RPCBind.Network(), config.RPCBind.String()); err != nil {
		return errors.Wrapf(err, "failed to bind agent to %s", config.RPCBind)
	}

	ctx.grpc("agent", server, bind)

	return nil
}
