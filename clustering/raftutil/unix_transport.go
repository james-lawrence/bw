package raftutil

import (
	"net"
	"path/filepath"
	"time"

	"github.com/hashicorp/memberlist"
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

// UnixAddressProvider - address provider for a unix stream layer.
type UnixAddressProvider struct {
	Dir string
}

// RaftAddr ...
func (t UnixAddressProvider) RaftAddr(n *memberlist.Node) (_zero raft.Server, err error) {
	return raft.Server{
		ID:       raft.ServerID(n.Name),
		Address:  raft.ServerAddress(filepath.Join(t.Dir, n.Name)),
		Suffrage: raft.Voter,
	}, err
}
