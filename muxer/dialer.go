package muxer

import (
	"context"
	"net"

	"github.com/hashicorp/yamux"
)

// Newdialer net.Dialer for the given protocol.
func NewDialer(protocol string, d net.Dialer) Dialer {
	return Dialer{
		p: Proto(protocol),
		d: d,
	}
}

// Dialer implements the net.Dialer interface
type Dialer struct {
	p Protocol
	d net.Dialer
}

func (t Dialer) Dial(network string, address string) (conn net.Conn, err error) {
	return t.DialContext(context.Background(), network, address)
}

func (t Dialer) DialContext(ctx context.Context, network string, address string) (conn net.Conn, err error) {
	var (
		session *yamux.Session
	)

	if conn, err = t.d.DialContext(ctx, network, address); err != nil {
		return conn, err
	}

	if session, err = yamux.Client(conn, nil); err != nil {
		conn.Close()
		return nil, err
	}

	if conn, err = session.Open(); err != nil {
		conn.Close()
		return nil, err
	}

	if err = handshakeOutbound(t.p[:], conn); err != nil {
		conn.Close()
		return nil, err
	}

	return conn, nil
}
