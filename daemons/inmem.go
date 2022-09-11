package daemons

import (
	"context"
	"net"
	"time"

	"github.com/akutz/memconn"
	"github.com/james-lawrence/bw"
	_cluster "github.com/james-lawrence/bw/cluster"
	"github.com/james-lawrence/bw/internal/envx"
	"github.com/james-lawrence/bw/internal/grpcx"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

func Inmem(dctx Context) (_ Context, err error) {
	const inmemconn = "bw.memu"
	var (
		l net.Listener
	)

	if l, err = memconn.Listen("memu", inmemconn); err != nil {
		return dctx, err
	}

	srv := grpc.NewServer(
		// grpc.UnaryInterceptor(grpcx.DebugIntercepter),
		// grpc.StreamInterceptor(grpcx.DebugStreamIntercepter),
		grpc.KeepaliveParams(dctx.RPCKeepalive),
		grpc.KeepaliveEnforcementPolicy(dctx.RPCKeepalivePolicy),
	)

	dctx.PeeringEvents.Bind(srv)

	dctx.grpc("inmem", srv, l)

	dialctx, done := context.WithTimeout(dctx.Context, time.Second)
	dctx.Inmem, err = grpc.DialContext(
		dialctx,
		inmemconn,
		grpc.WithTransportCredentials(insecure.NewCredentials()),
		grpc.WithBlock(),
		grpcx.DialInmem(),
	)
	done()
	if err != nil {
		return dctx, err
	}

	if envx.Boolean(false, bw.EnvLogsVerbose) {
		if err = _cluster.NewEventsSubscription(dctx.Context, dctx.Inmem, _cluster.LoggingSubscription); err != nil {
			return dctx, err
		}
	}

	return dctx, err
}
