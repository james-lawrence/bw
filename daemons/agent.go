package daemons

import (
	"net"
	"path/filepath"
	"time"

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
	"google.golang.org/grpc/keepalive"
)

// Agent daemon - rpc endpoint for the system.
func Agent(ctx Context, cx cluster, config agent.Config) (err error) {
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

	keepalive := grpc.KeepaliveParams(keepalive.ServerParameters{
		MaxConnectionIdle: 1 * time.Hour,
		Time:              1 * time.Minute,
		Timeout:           2 * time.Minute,
	})

	dialer := agent.NewDialer(agent.DefaultDialerOptions(grpc.WithTransportCredentials(ctx.RPCCredentials))...)
	qdialer := agent.NewQuorumDialer(dialer)
	dispatcher := agentutil.NewDispatcher(cx, qdialer)

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

	q := quorum.New(
		observersdir,
		cx,
		proxy.NewProxy(cx),
		ctx.Upload,
		quorum.OptionDialer(dialer),
	)
	go (&q).Observe(ctx.Raft, make(chan raft.Observation, 200))

	a := agent.NewServer(
		cx,
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

	ctx.grpc(server, bind)

	return nil
}
