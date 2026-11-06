package muxer

import (
	"crypto/md5"
	"crypto/tls"
)

// Protocol uuid.
type Protocol [16]byte

func Proto(name string) Protocol {
	return md5.Sum([]byte(name))
}

// tlsState is satisfied by *tls.Conn and by stream connections of transports which
// secure the connection with tls (quic) so the negotiated protocol (alpn) can be checked.
type tlsState interface {
	ConnectionState() tls.ConnectionState
}

// tlsConn is a tls connection that may need to complete its handshake.
type tlsConn interface {
	tlsState
	Handshake() error
}
