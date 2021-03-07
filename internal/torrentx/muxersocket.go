package torrentx

import (
	"context"
	"log"
	"net"
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
	log.Println("torrentx.Socket Dial", addr)
	return t.Dialer.DialContext(ctx, t.Listener.Addr().Network(), addr)
}
