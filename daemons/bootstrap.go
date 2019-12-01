package daemons

import (
	"context"
	"crypto/tls"

	"github.com/james-lawrence/bw/agent"
	"github.com/james-lawrence/bw/bootstrap"
	"github.com/james-lawrence/bw/storage"

	"github.com/pkg/errors"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials"
)

// Bootstrap daemon - pulls latest deploy from the cluster and ensures its running locally.
func Bootstrap(ctx context.Context, cx cluster, config agent.Config, dl storage.DownloadProtocol) (err error) {
	var (
		creds *tls.Config
	)

	if creds, err = config.BuildServer(); err != nil {
		return err
	}

	dialer := agent.NewDialer(
		agent.DefaultDialerOptions(
			grpc.WithTransportCredentials(credentials.NewTLS(creds)),
		)...,
	)

	bootstrap.CleanSockets(config)

	if err = bootstrap.NewLocal(cx.Local(), dialer).Bind(ctx, bootstrap.SocketLocal(config)); err != nil {
		return errors.Wrap(err, "failed to initialize local bootstrap service")
	}

	if err = bootstrap.NewQuorum(cx, dialer).Bind(ctx, bootstrap.SocketQuorum(config)); err != nil {
		return errors.Wrap(err, "failed to initialize quorum bootstrap service")
	}

	if err = bootstrap.NewCluster(cx, dialer).Bind(ctx, bootstrap.SocketAuto(config)); err != nil {
		return errors.Wrap(err, "failed to initialize cluster bootstrap service")
	}

	if err = bootstrap.NewFilesystem(config, cx, dialer).Bind(ctx, bootstrap.SocketAuto(config)); err != nil {
		return errors.Wrap(err, "failed to initialize filesystem bootstrap service")
	}

	bus := bootstrap.NewUntilSuccess(
		bootstrap.OptionMaxAttempts(config.Bootstrap.Attempts),
	)

	if err = bus.Run(ctx, config, dl); err != nil {
		// if bootstrapping fails shutdown the process.
		return errors.Wrap(err, "failed to bootstrap node shutting down")
	}

	return nil
}
