package muxer

import (
	"io"
	"net"
	sync "sync"
)

func newListener(m *M, addr net.Addr, p Protocol) listener {
	return listener{
		p:        p,
		m:        m,
		inbound:  make(chan net.Conn),
		shutdown: &sync.Once{},
		addr:     addr,
	}
}

type listener struct {
	p        Protocol
	m        *M
	inbound  chan net.Conn
	shutdown *sync.Once
	addr     net.Addr
}

func (t listener) Accept() (c net.Conn, err error) {
	if c, ok := <-t.inbound; ok {
		return c, nil
	} else {
		return nil, io.EOF
	}
}

func (t listener) Close() error {
	t.shutdown.Do(func() {
		close(t.inbound)
		t.m.release(t.p)
	})
	return nil
}

func (t listener) Addr() net.Addr {
	return t.addr
}
