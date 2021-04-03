package dialers

import (
	"context"
	"math/rand"
	"net"
	"net/url"
	"time"

	"github.com/hashicorp/memberlist"
	"github.com/james-lawrence/bw/agent"
	"github.com/james-lawrence/bw/internal/x/grpcx"
	"github.com/james-lawrence/bw/internal/x/tlsx"
	"github.com/james-lawrence/bw/muxer"
	"google.golang.org/grpc"
)

type dialer interface {
	DialContext(ctx context.Context, network string, address string) (net.Conn, error)
}

type rendezvous interface {
	GetN(n int, key []byte) []*memberlist.Node
}

// Dialer the interface for dialing the cluster.
type Dialer interface {
	Dial(options ...grpc.DialOption) (c *grpc.ClientConn, err error)
}

// ContextDialer the interface for dialing the cluster.
type ContextDialer interface {
	DialContext(ctx context.Context, options ...grpc.DialOption) (c *grpc.ClientConn, err error)
}

// Defaults return a set of default dialing options.
// accepts additional options to merge in.
// this allows for converting from one dialer to another.
type Defaults interface {
	Defaults(combined ...grpc.DialOption) []grpc.DialOption
}

// DefaultsDialer combines both defaults and dialer.
type DefaultsDialer interface {
	ContextDialer
	Defaults
}

// DefaultDialerOptions sets reasonable defaults for dialing the agent.
func DefaultDialerOptions(options ...grpc.DialOption) (results []grpc.DialOption) {
	defaults := []grpc.DialOption{
		grpc.WithBackoffMaxDelay(5 * time.Second),
		grpc.WithUnaryInterceptor(grpcx.DebugClientIntercepter),
		grpc.WithStreamInterceptor(grpcx.DebugClientStreamIntercepter),
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
func WithMuxer(d *tlsx.Dialer, n net.Addr) grpc.DialOption {
	return grpc.WithContextDialer(func(ctx context.Context, address string) (conn net.Conn, err error) {
		proto, host, err := parseURI(address)
		if err != nil {
			return nil, err
		}

		return muxer.NewDialer(proto, d).DialContext(ctx, n.Network(), host)
	})
}

func parseURI(s string) (p string, host string, err error) {
	uri, err := url.Parse(s)
	if err != nil {
		return p, host, err
	}

	return uri.Scheme, uri.Host, nil
}
