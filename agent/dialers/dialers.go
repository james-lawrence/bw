package dialers

import (
	"context"
	"math/rand"
	"net"
	"time"

	"github.com/hashicorp/memberlist"
	"github.com/james-lawrence/bw/agent"
	"github.com/james-lawrence/bw/muxer"
	"google.golang.org/grpc"
	"google.golang.org/grpc/backoff"
)

type dialer interface {
	DialContext(ctx context.Context, network string, address string) (net.Conn, error)
}

type rendezvous interface {
	GetN(n int, key []byte) []*memberlist.Node
}

// ContextDialer the interface for dialing the cluster.
type ContextDialer interface {
	DialContext(ctx context.Context, options ...grpc.DialOption) (c *grpc.ClientConn, err error)
}

// Defaults return a set of default dialing options.
// accepts additional options to merge in.
// this allows for converting from one dialer to another.
type Defaults interface {
	Defaults(combined ...grpc.DialOption) Defaulted
}

// DefaultsDialer combines both defaults and dialer.
type DefaultsDialer interface {
	ContextDialer
	Defaults
}

// DefaultDialerOptions sets reasonable defaults for dialing the agent.
func DefaultDialerOptions(options ...grpc.DialOption) (results Defaulted) {
	boff := backoff.DefaultConfig
	boff.MaxDelay = 5 * time.Second

	defaults := []grpc.DialOption{
		grpc.WithConnectParams(grpc.ConnectParams{
			Backoff: boff,
		}),
		// grpc.WithUnaryInterceptor(grpcx.DebugClientIntercepter),
		// grpc.WithStreamInterceptor(grpcx.DebugClientStreamIntercepter),
	}

	return append(
		defaults,
		options...,
	)
}

func shuffleQuorum(q []*agent.Peer) []*agent.Peer {
	rand.Shuffle(len(q), func(i int, j int) {
		q[i], q[j] = q[j], q[i]
	})
	return q
}

// WithMuxer dialer to connect using a connection muxer.
func WithMuxer(d dialer, n net.Addr) grpc.DialOption {
	return grpc.WithContextDialer(func(ctx context.Context, address string) (conn net.Conn, err error) {
		proto, host, err := muxer.ParseURI(address)
		if err != nil {
			return nil, err
		}

		return muxer.NewDialer(proto, d).DialContext(ctx, n.Network(), host)
	})
}
