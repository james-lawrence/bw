package muxer

import (
	"context"
	"crypto/tls"
	"net"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

type fakeTLS struct {
	net.Conn
	proto string
}

func (t fakeTLS) Handshake() error { return nil }
func (t fakeTLS) ConnectionState() tls.ConnectionState {
	return tls.ConnectionState{NegotiatedProtocol: t.proto}
}

// a connection which reports a tls state without negotiating bw.mux must not reach the protocol handshake.
func TestAcceptRejectsUnnegotiatedALPN(t *testing.T) {
	a, b := net.Pipe()
	defer b.Close()
	m := New()

	require.NoError(t, accept(context.Background(), m, fakeTLS{Conn: a, proto: "h2"}))

	b.SetReadDeadline(time.Now().Add(time.Second))
	_, err := b.Read(make([]byte, 1))
	require.Error(t, err, "connection should have been closed")
}
