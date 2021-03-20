package muxer

import (
	"net"

	"github.com/hashicorp/yamux"
)

type option func(*M) error

func New(options ...option) *M {
	return &M{
		m: &sync.RWMutex{},
	}
}

type M struct {
	m         *sync.RWMutex
	listeners []net.Listener
}

func (M) listen(l net.Listener) error {
	for {
		var (
			err  error
			conn net.Conn
		)

		if conn, err = l.Accept(); err != nil {
			return err
		}

		if session, err := yamux.Server(conn, nil); err != nil {
			return err
		}

	}

	return nil
}
