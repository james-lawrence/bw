package daemons

import (
	"context"
	"log"
	"net"
	"time"

	"github.com/akutz/memconn"
	"github.com/james-lawrence/bw"
	"github.com/james-lawrence/bw/agent"
	_cluster "github.com/james-lawrence/bw/cluster"
	"github.com/james-lawrence/bw/internal/x/envx"
	"github.com/james-lawrence/bw/internal/x/grpcx"
	"github.com/james-lawrence/bw/notary"
	"google.golang.org/grpc"
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
		grpc.UnaryInterceptor(grpcx.DebugIntercepter),
		grpc.StreamInterceptor(grpcx.DebugStreamIntercepter),
		grpc.KeepaliveParams(dctx.RPCKeepalive),
	)

	dctx.PeeringEvents.Bind(srv)

	dctx.grpc("inmem", srv, l)

	dctx.Inmem, err = grpc.DialContext(dctx.Context, inmemconn, grpc.WithInsecure(), grpc.WithContextDialer(func(ctx context.Context, address string) (conn net.Conn, err error) {
		ctx, done := context.WithTimeout(ctx, time.Second)
		defer done()
		return memconn.DialContext(ctx, "memu", address)
	}))

	if err != nil {
		return dctx, err
	}

	if err = _cluster.NewEventsSubscription(dctx.Inmem, _cluster.LoggingSubscription); err != nil {
		return dctx, err
	}

	// Notary Subscriptions to node events. tracks the public key signatures for nodes in the cluster.
	err = _cluster.NewEventsSubscription(dctx.Inmem, func(ctx context.Context, evt *agent.ClusterWatchEvents) (err error) {
		if len(evt.Node.PublicKey) == 0 {
			if envx.Boolean(false, bw.EnvLogsVerbose) {
				log.Println("Notary.Subscription ignoring event - no public key", evt.Event.String(), evt.Node.Ip, evt.Node.Name)
			}
			return nil
		}

		if envx.Boolean(false, bw.EnvLogsVerbose) {
			log.Println("Notary.Subscription", evt.Event.String(), evt.Node.Ip, evt.Node.Name)
		}

		switch evt.Event {
		case agent.ClusterWatchEvents_Joined, agent.ClusterWatchEvents_Update:
			_, err = dctx.NotaryStorage.Insert(notary.AgentGrant(evt.Node.PublicKey))
			return err
		case agent.ClusterWatchEvents_Depart:
			_, err = dctx.NotaryStorage.Delete(notary.AgentGrant(evt.Node.PublicKey))
			return err
		default:
			return nil
		}
	})

	return dctx, err
}
