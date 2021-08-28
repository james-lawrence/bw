package daemons

import (
	"log"
	"net"

	"github.com/pkg/errors"
	"google.golang.org/grpc"

	"github.com/james-lawrence/bw"
	"github.com/james-lawrence/bw/agent/acme"
)

// Autocert - used to bootstrap certificates.
func Autocert(dctx Context) (err error) {
	var (
		bind net.Listener
	)

	server := grpc.NewServer(
		// grpc.UnaryInterceptor(grpcx.DebugIntercepter),
		// grpc.StreamInterceptor(grpcx.DebugStreamIntercepter),
		grpc.KeepaliveParams(dctx.RPCKeepalive),
		grpc.KeepaliveEnforcementPolicy(dctx.RPCKeepalivePolicy),
	)
	acme.RegisterACMEServer(server, acme.NewService(dctx.ACMECache, dctx.NotaryAuth))

	log.Println("autocert bind", bw.ProtocolAutocert, dctx.Listener.Addr().String())
	if bind, err = dctx.Muxer.Bind(bw.ProtocolAutocert, dctx.Listener.Addr()); err != nil {
		return errors.Wrap(err, "failed to bind autocert service")
	}

	dctx.grpc("autocert", server, bind)

	return nil
}
