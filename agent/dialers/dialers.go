package dialers

import (
	"context"
	"net"
	"time"

	"github.com/hashicorp/memberlist"
	"github.com/james-lawrence/bw/muxer"
	"google.golang.org/grpc"
	"google.golang.org/grpc/backoff"
)

// Define a private type for your context keys to avoid collisions across packages
type contextKey string

const peerKey contextKey = "peer_address"

// Attach a value
func WithAddress(ctx context.Context, pstring string) context.Context {
	return context.WithValue(ctx, peerKey, pstring)
}

func Address(ctx context.Context) string {
	p, _ := ctx.Value(peerKey).(string)
	return p
}

type dialer interface {
	DialContext(ctx context.Context, network string, address string) (net.Conn, error)
}

type rendezvous interface {
	GetN(n int, key []byte) []*memberlist.Node
}

// T the interface for dialing the cluster.
type T interface {
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
	T
	Defaults
}

// DefaultDialerOptions sets reasonable defaults for dialing the agent.
func DefaultDialerOptions(options ...grpc.DialOption) (results Defaulted) {
	boff := backoff.DefaultConfig
	boff.MaxDelay = 5 * time.Second

	defaults := []grpc.DialOption{
		grpc.WithConnectParams(grpc.ConnectParams{
			Backoff:           boff,
			MinConnectTimeout: 20 * time.Second,
		}),
	}
	return NewDefaults(defaults...).Defaults(options...)
}

// WithMuxer dialer to connect using a connection muxer.
func WithMuxer(d dialer, n net.Addr) grpc.DialOption {
	return grpc.WithContextDialer(func(ctx context.Context, address string) (conn net.Conn, err error) {
		proto, host, err := muxer.ParseURI(address)
		if err != nil {
			return nil, err
		}

		conn, err = muxer.NewDialer(proto, d).DialContext(ctx, n.Network(), host)
		return wrapconn(conn, address), err
	})
}

type remoteconn struct {
	net.Conn
	remote net.Addr
}

// wrapconn wraps a net.Conn and overrides its RemoteAddr return value.
func wrapconn(conn net.Conn, address string) net.Conn {
	if conn == nil {
		return nil
	}

	return &remoteconn{
		Conn:   conn,
		remote: newagentaddr(conn.RemoteAddr(), address),
	}
}

func (c *remoteconn) RemoteAddr() net.Addr {
	if c.remote != nil {
		return c.remote
	}
	return c.Conn.RemoteAddr()
}

type agentaddr struct {
	o       net.Addr
	address string
}

func newagentaddr(o net.Addr, address string) agentaddr {
	return agentaddr{
		o:       o,
		address: address,
	}
}

func (a agentaddr) Network() string {
	return a.o.Network()
}

func (a agentaddr) String() string {
	return a.address
}
