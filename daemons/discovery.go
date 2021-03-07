package daemons

import (
	"log"
	"net"

	"github.com/pkg/errors"
	"google.golang.org/grpc"

	"github.com/james-lawrence/bw"
	"github.com/james-lawrence/bw/agent/discovery"
	"github.com/james-lawrence/bw/internal/x/grpcx"
)

// Discovery initiates the discovery backend.
func Discovery(ctx Context) (err error) {
	var (
		// deprecatedbind net.Listener
		bind   net.Listener
		server *grpc.Server
	)

	server = grpc.NewServer(
		grpc.UnaryInterceptor(grpcx.DebugIntercepter),
		grpc.StreamInterceptor(grpcx.DebugStreamIntercepter),
		grpc.KeepaliveParams(ctx.RPCKeepalive),
	)

	// dialer := dialers.NewQuorum(
	// 	ctx.Cluster,
	// 	ctx.Dialer.Defaults()...,
	// )

	// notary.NewProxy(dialer).Bind(server)

	// exposes details about the cluster.
	discovery.New(ctx.Cluster).Bind(server)

	// used to validate client certificates.
	// discovery.NewAuthority(
	// 	certificatecache.CAKeyPath(ctx.Config.CredentialsDir, certificatecache.DefaultTLSGeneratedCAProto),
	// ).Bind(server)

	// log.Println("discovery", ctx.Config.DiscoveryBind.String())
	// if bind, err = net.Listen(ctx.Config.DiscoveryBind.Network(), ctx.Config.DiscoveryBind.String()); err != nil {
	// 	return errors.Wrapf(err, "failed to bind discovery to %s", ctx.Config.DiscoveryBind)
	// }

	log.Printf("discovery: %T %s", ctx.Listener, ctx.Listener.Addr().String())
	if bind, err = ctx.Muxer.Bind(bw.ProtocolDiscovery, ctx.Listener.Addr()); err != nil {
		return errors.Wrapf(err, "failed to bind discovery to %s", ctx.Listener.Addr().String())
	}
	ctx.grpc("discovery", server, bind)

	return nil
}
