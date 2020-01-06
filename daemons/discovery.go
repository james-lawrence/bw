package daemons

import (
	"crypto/tls"
	"net"

	"github.com/pkg/errors"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials"

	"github.com/james-lawrence/bw/agent"
	"github.com/james-lawrence/bw/agent/acme"
	"github.com/james-lawrence/bw/agent/dialers"
	"github.com/james-lawrence/bw/agent/discovery"
	"github.com/james-lawrence/bw/agent/proxy"
	"github.com/james-lawrence/bw/certificatecache"
	"github.com/james-lawrence/bw/internal/x/tlsx"
	"github.com/james-lawrence/bw/notary"
)

// Discovery initiates the discovery backend.
func Discovery(ctx Context) (err error) {
	var (
		bind      net.Listener
		tlsconfig *tls.Config
		server    *grpc.Server
	)

	keepalive := grpc.KeepaliveParams(ctx.RPCKeepalive)

	if tlsconfig, err = TLSGenServer(ctx.Config, tlsx.OptionVerifyClientIfGiven); err != nil {
		return err
	}
	tlsconfig = certificatecache.NewALPN(tlsconfig, acme.NewALPNCertCache(acme.NewClient(ctx.Cluster)))

	if bind, err = net.Listen(ctx.Config.DiscoveryBind.Network(), ctx.Config.DiscoveryBind.String()); err != nil {
		return errors.Wrapf(err, "failed to bind discovery to %s", ctx.Config.DiscoveryBind)
	}

	server = grpc.NewServer(
		// grpc.UnaryInterceptor(grpcx.DebugIntercepter),
		// grpc.StreamInterceptor(grpcx.DebugStreamIntercepter),
		grpc.Creds(credentials.NewTLS(tlsconfig)),
		keepalive,
	)

	dialer := dialers.NewQuorum(
		ctx.Cluster,
		agent.DefaultDialerOptions(
			grpc.WithTransportCredentials(ctx.GRPCCreds()),
		)...,
	)

	proxy.NewDeployment(notary.NewAuth(ctx.NotaryStorage), dialer).Bind(server)
	notary.NewProxy(dialer).Bind(server)
	discovery.New(ctx.Cluster).Bind(server)

	// used to validate client certificates.
	discovery.NewAuthority(
		certificatecache.CAKeyPath(ctx.Config.CredentialsDir, certificatecache.DefaultTLSGeneratedCAProto),
	).Bind(server)

	ctx.grpc("discovery", server, bind)

	return nil
}
