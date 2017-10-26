package raftutil

import (
	"crypto/tls"
	"net"
	"time"

	"github.com/hashicorp/raft"
	"github.com/pkg/errors"
)

// NewTLSStreamLayer ...
func NewTLSStreamLayer(l net.Listener, cs *tls.Config) StreamLayer {
	l = tls.NewListener(l, cs)
	return StreamLayer{
		Listener: l,
		c:        cs,
	}
}

// StreamLayer ...
type StreamLayer struct {
	net.Listener
	c *tls.Config
}

// Dial is used to create a new outgoing connection
func (t StreamLayer) Dial(address raft.ServerAddress, timeout time.Duration) (conn net.Conn, err error) {
	d := &net.Dialer{
		Timeout:   timeout,
		KeepAlive: 5 * time.Second,
	}

	if conn, err = tls.DialWithDialer(d, t.Listener.Addr().Network(), string(address), t.c); err != nil {
		return conn, errors.WithStack(err)
	}

	return conn, nil
}
