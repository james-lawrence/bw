package raftutil

import (
	"net"
	"time"

	"github.com/hashicorp/raft"
	"github.com/pkg/errors"
)

// NewUnixStreamLayer ...
func NewUnixStreamLayer(l net.Listener) UnixStreamLayer {
	return UnixStreamLayer{
		Listener: l,
	}
}

// UnixStreamLayer ...
type UnixStreamLayer struct {
	net.Listener
}

// Dial is used to create a new outgoing connection
func (t UnixStreamLayer) Dial(address raft.ServerAddress, timeout time.Duration) (conn net.Conn, err error) {
	addr := net.UnixAddr{Net: "unix", Name: string(address)}
	if conn, err = net.DialUnix("unix", nil, &addr); err != nil {
		return conn, errors.WithStack(err)
	}

	return conn, nil
}
