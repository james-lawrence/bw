package daemons

import (
	"log"
	"net"

	"github.com/pkg/errors"
	"google.golang.org/grpc"

	"github.com/james-lawrence/bw"
	"github.com/james-lawrence/bw/agent/acme"
	"github.com/james-lawrence/bw/internal/x/grpcx"
)

// Autocert - used to bootstrap certificates.
func Autocert(ctx Context) (err error) {
	var (
		bind net.Listener
	)

	server := grpc.NewServer(
		grpc.UnaryInterceptor(grpcx.DebugIntercepter),
		grpc.StreamInterceptor(grpcx.DebugStreamIntercepter),
		grpc.KeepaliveParams(ctx.RPCKeepalive),
	)
	acme.RegisterACMEServer(server, acme.NewService(ctx.ACMECache, ctx.NotaryAuth))

	log.Println("autocert bind", bw.ProtocolAutocert, ctx.Listener.Addr().String())
	if bind, err = ctx.Muxer.Bind(bw.ProtocolAutocert, ctx.Listener.Addr()); err != nil {
		return errors.Wrap(err, "failed to bind autocert service")
	}

	ctx.grpc("autocert", server, bind)

	return nil
}
