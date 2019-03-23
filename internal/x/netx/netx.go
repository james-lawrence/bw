package netx

import (
	"fmt"
	"net"
)

// NewNoopListener -- junk listener.
func NewNoopListener() net.Listener {
	return noopListener{}
}

type noopListener struct{}

func (noopListener) Accept() (net.Conn, error) {
	return nil, fmt.Errorf("noopListener can not accept connections")
}

func (noopListener) Close() error {
	return nil
}

func (noopListener) Addr() net.Addr {
	return &net.UnixAddr{
		Name: "foo",
		Net:  "unix",
	}
}
