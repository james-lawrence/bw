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
	return StreamLayer{
		Listener: tls.NewListener(l, cs),
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

// NewTLSTCP StreamLayer
func NewTLSTCP(s string, cs *tls.Config) (sl StreamLayer, err error) {
	var (
		addr *net.TCPAddr
		l    net.Listener
	)

	if addr, err = net.ResolveTCPAddr("tcp", s); err != nil {
		return sl, err
	}

	if l, err = net.ListenTCP(addr.Network(), addr); err != nil {
		return sl, errors.WithStack(err)
	}

	return NewTLSStreamLayer(l, cs), nil
}
