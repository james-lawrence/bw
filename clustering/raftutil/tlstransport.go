package raftutil

import (
	"context"
	"crypto/tls"
	"net"
	"time"

	"github.com/hashicorp/raft"
	"github.com/james-lawrence/bw/internal/x/tlsx"
	"github.com/pkg/errors"
)

type dialer interface {
	DialContext(ctx context.Context, network string, address string) (net.Conn, error)
}

func NewTLSStreamDialer(cs *tls.Config) dialer {
	if cs == nil {
		return &net.Dialer{
			KeepAlive: 5 * time.Second,
		}
	}

	return &tlsx.Dialer{
		NetDialer: &net.Dialer{
			KeepAlive: 5 * time.Second,
		},
		Config: cs,
	}
}

// NewStreamTransport ...
func NewStreamTransport(l net.Listener, d dialer) StreamLayer {
	return StreamLayer{
		d:        d,
		Listener: l,
	}
}

// StreamLayer ...
type StreamLayer struct {
	d dialer
	net.Listener
}

// Dial is used to create a new outgoing connection
func (t StreamLayer) Dial(address raft.ServerAddress, timeout time.Duration) (conn net.Conn, err error) {
	ctx, done := context.WithTimeout(context.Background(), timeout)
	defer done()

	if conn, err = t.d.DialContext(ctx, t.Listener.Addr().Network(), string(address)); err != nil {
		return conn, errors.WithStack(err)
	}

	return conn, nil
}
