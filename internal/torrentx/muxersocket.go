package torrentx

import (
	"context"
	"net"

	"github.com/pkg/errors"
)

type dialer interface {
	DialContext(ctx context.Context, network, address string) (net.Conn, error)
}

// Socket ...
type Socket struct {
	net.Listener
	Dialer dialer
}

// Dial the given address
func (t Socket) Dial(ctx context.Context, addr string) (net.Conn, error) {
	if addr == t.Listener.Addr().String() {
		return nil, errors.Errorf("attempted to dial self: %s -> %s", addr, t.Listener.Addr().String())
	}

	return t.Dialer.DialContext(ctx, t.Listener.Addr().Network(), addr)
}
