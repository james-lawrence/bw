package daemons

import (
	"log"
	"net"

	"github.com/pkg/errors"
	"google.golang.org/grpc"

	"github.com/james-lawrence/bw"
	"github.com/james-lawrence/bw/agent/discovery"
	"github.com/james-lawrence/bw/certificatecache"
	"github.com/james-lawrence/bw/internal/x/grpcx"
)

// Discovery initiates the discovery backend.
func Discovery(ctx Context) (err error) {
	var (
		bind   net.Listener
		server *grpc.Server
	)

	server = grpc.NewServer(
		grpc.UnaryInterceptor(grpcx.DebugIntercepter),
		grpc.StreamInterceptor(grpcx.DebugStreamIntercepter),
		grpc.KeepaliveParams(ctx.RPCKeepalive),
	)

	// exposes details about the cluster.
	discovery.New(ctx.Cluster).Bind(server)

	// used to validate client certificates.
	discovery.NewAuthority(
		certificatecache.CAKeyPath(ctx.Config.CredentialsDir, certificatecache.DefaultTLSGeneratedCAProto),
	).Bind(server)

	log.Printf("discovery: %T %s", ctx.Listener, ctx.Listener.Addr().String())
	if bind, err = ctx.Muxer.Bind(bw.ProtocolDiscovery, ctx.Listener.Addr()); err != nil {
		return errors.Wrapf(err, "failed to bind discovery to %s", ctx.Listener.Addr().String())
	}
	ctx.grpc("discovery", server, bind)

	return nil
}
