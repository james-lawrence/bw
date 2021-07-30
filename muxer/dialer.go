package muxer

import (
	"context"
	"crypto/tls"
	"log"
	"net"
	"net/url"
	"sync/atomic"

	"github.com/pkg/errors"
)

type dialer interface {
	DialContext(ctx context.Context, network string, address string) (net.Conn, error)
}

var i = new(int64)

// ParseURI parse an address in the form:
// protocol://host:port
func ParseURI(s string) (p string, host string, err error) {
	uri, err := url.Parse(s)
	if err != nil {
		return p, host, err
	}

	return uri.Scheme, uri.Host, nil
}

// Newdialer net.Dialer for the given protocol.
func NewDialer(protocol string, d dialer) Dialer {
	return Dialer{
		id:       atomic.AddInt64(i, 1),
		protocol: protocol,
		digest:   Proto(protocol),
		d:        d,
	}
}

// Dialer implements the net.Dialer interface
type Dialer struct {
	id       int64
	protocol string
	digest   Protocol
	d        dialer
}

func (t Dialer) Dial(network string, address string) (conn net.Conn, err error) {
	// log.Printf("muxer.Dial initiated: %T %s %s %s\n", t.d, t.protocol, network, address)
	// defer log.Printf("muxer.Dial completed: %T %s %s %s\n", t.d, t.protocol, network, address)
	return t.DialContext(context.Background(), network, address)
}

func (t Dialer) DialContext(ctx context.Context, network string, address string) (conn net.Conn, err error) {
	type handshaker interface {
		Handshake() error
	}
	// log.Printf("muxer.DialContext initiated: %T %s %s %s\n", t.d, t.protocol, network, address)
	// defer log.Printf("muxer.DialContext completed: %T %s %s %s\n", t.d, t.protocol, network, address)

	if conn, err = t.d.DialContext(ctx, network, address); err != nil {
		return conn, errors.Wrapf(err, "muxer.DialContext failed: %s %s://%s", t.protocol, network, address)
	}

	if c, ok := conn.(handshaker); ok {
		if err := c.Handshake(); err != nil {
			conn.Close()
			return nil, errors.Wrap(err, "handshake failed")
		}
	}

	if tlsconn, ok := conn.(*tls.Conn); ok {
		s := tlsconn.ConnectionState()
		if s.NegotiatedProtocol != "bw.mux" {
			return conn, nil
		}
	}

	if err = handshakeOutbound(t.digest[:], conn); err != nil {
		log.Println("muxer.DialContext handshakeOutbound", t.protocol, network, address, err)
		conn.Close()
		return nil, err
	}

	return conn, nil
}
