package daemons

import (
	"github.com/james-lawrence/bw/agent"
	"github.com/james-lawrence/bw/bootstrap"

	"github.com/pkg/errors"
	"google.golang.org/grpc"
)

// Bootstrap daemon - pulls latest deploy from the cluster and ensures its running locally.
func Bootstrap(ctx Context) (err error) {
	cx := ctx.Cluster
	dialer := agent.NewDialer(
		agent.DefaultDialerOptions(
			grpc.WithTransportCredentials(ctx.GRPCCreds()),
		)...,
	)

	bootstrap.CleanSockets(ctx.Config)

	if err = bootstrap.NewLocal(cx.Local(), dialer).Bind(ctx.Context, bootstrap.SocketLocal(ctx.Config)); err != nil {
		return errors.Wrap(err, "failed to initialize local bootstrap service")
	}

	if err = bootstrap.NewQuorum(cx, dialer).Bind(ctx.Context, bootstrap.SocketQuorum(ctx.Config)); err != nil {
		return errors.Wrap(err, "failed to initialize quorum bootstrap service")
	}

	if err = bootstrap.NewCluster(cx, dialer).Bind(ctx.Context, bootstrap.SocketAuto(ctx.Config)); err != nil {
		return errors.Wrap(err, "failed to initialize cluster bootstrap service")
	}

	if err = bootstrap.NewFilesystem(ctx.Config, cx, dialer).Bind(ctx.Context, bootstrap.SocketAuto(ctx.Config)); err != nil {
		return errors.Wrap(err, "failed to initialize filesystem bootstrap service")
	}

	bus := bootstrap.NewUntilSuccess(
		bootstrap.OptionMaxAttempts(ctx.Config.Bootstrap.Attempts),
	)

	if err = bus.Run(ctx.Context, ctx.Config, ctx.Download); err != nil {
		// if bootstrapping fails shutdown the process.
		return errors.Wrap(err, "failed to bootstrap node shutting down")
	}

	return nil
}
