package notary

import (
	"context"
	"sync"

	"google.golang.org/grpc"
)

func maybeNotaryClient(cc *grpc.ClientConn, err error) (NotaryClient, error) {
	if err != nil {
		return nil, err
	}

	return NewNotaryClient(cc), nil
}

func newCached(d dialer) cached {
	return cached{
		dialer: d,
		m:      &sync.RWMutex{},
	}
}

type cached struct {
	dialer
	conn *grpc.ClientConn
	m    *sync.RWMutex
}

func (t cached) cached() (cc *grpc.ClientConn, err error) {
	t.m.RLock()
	cc = t.conn
	t.m.RUnlock()

	if cc != nil {
		return cc, nil
	}

	t.m.Lock()
	defer t.m.Unlock()

	if t.conn, err = t.dialer.DialContext(context.Background()); err != nil {
		return nil, err
	}

	return t.conn, nil
}
