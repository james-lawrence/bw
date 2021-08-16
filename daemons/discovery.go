package daemons

import (
	"log"
	"net"

	"github.com/pkg/errors"
	"google.golang.org/grpc"

	"github.com/james-lawrence/bw"
	"github.com/james-lawrence/bw/agent/discovery"
	"github.com/james-lawrence/bw/certificatecache"
)

// Discovery initiates the discovery backend.
func Discovery(dctx Context) (err error) {
	var (
		bind   net.Listener
		server *grpc.Server
	)

	server = grpc.NewServer(
		// grpc.UnaryInterceptor(grpcx.DebugIntercepter),
		// grpc.StreamInterceptor(grpcx.DebugStreamIntercepter),
		grpc.KeepaliveParams(dctx.RPCKeepalive),
	)

	// exposes details about the cluster.
	discovery.New(dctx.Cluster).Bind(server)

	// used to validate client certificates.
	discovery.NewAuthority(
		certificatecache.CAKeyPath(dctx.Config.CredentialsDir, certificatecache.DefaultTLSGeneratedCAProto),
	).Bind(server)

	log.Printf("discovery: %T %s", dctx.Listener, dctx.Listener.Addr().String())
	if bind, err = dctx.Muxer.Bind(bw.ProtocolDiscovery, dctx.Listener.Addr()); err != nil {
		return errors.Wrapf(err, "failed to bind discovery to %s", dctx.Listener.Addr().String())
	}
	dctx.grpc("discovery", server, bind)

	return nil
}
