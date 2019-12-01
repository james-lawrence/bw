package daemons

import (
	"github.com/james-lawrence/bw/agent"
	"github.com/james-lawrence/bw/bootstrap"

	"github.com/pkg/errors"
	"google.golang.org/grpc"
)

// Bootstrap daemon - pulls latest deploy from the cluster and ensures its running locally.
func Bootstrap(ctx Context, cx cluster, config agent.Config) (err error) {
	dialer := agent.NewDialer(
		agent.DefaultDialerOptions(
			grpc.WithTransportCredentials(ctx.RPCCredentials),
		)...,
	)

	bootstrap.CleanSockets(config)

	if err = bootstrap.NewLocal(cx.Local(), dialer).Bind(ctx.Context, bootstrap.SocketLocal(config)); err != nil {
		return errors.Wrap(err, "failed to initialize local bootstrap service")
	}

	if err = bootstrap.NewQuorum(cx, dialer).Bind(ctx.Context, bootstrap.SocketQuorum(config)); err != nil {
		return errors.Wrap(err, "failed to initialize quorum bootstrap service")
	}

	if err = bootstrap.NewCluster(cx, dialer).Bind(ctx.Context, bootstrap.SocketAuto(config)); err != nil {
		return errors.Wrap(err, "failed to initialize cluster bootstrap service")
	}

	if err = bootstrap.NewFilesystem(config, cx, dialer).Bind(ctx.Context, bootstrap.SocketAuto(config)); err != nil {
		return errors.Wrap(err, "failed to initialize filesystem bootstrap service")
	}

	bus := bootstrap.NewUntilSuccess(
		bootstrap.OptionMaxAttempts(config.Bootstrap.Attempts),
	)

	if err = bus.Run(ctx.Context, config, ctx.Download); err != nil {
		// if bootstrapping fails shutdown the process.
		return errors.Wrap(err, "failed to bootstrap node shutting down")
	}

	return nil
}
