package dialers

import (
	"context"
	"sync"

	"google.golang.org/grpc"
	"google.golang.org/grpc/connectivity"
)

func NewCached(d DefaultsDialer) *Cached {
	return &Cached{
		d: d,
		m: &sync.RWMutex{},
	}
}

type Cached struct {
	d    DefaultsDialer
	conn *grpc.ClientConn
	m    *sync.RWMutex
}

func (t *Cached) Close() error {
	t.m.Lock()
	c := t.conn
	t.conn = nil
	t.m.Unlock()

	if c == nil {
		return nil
	}

	return c.Close()
}

func (t *Cached) DialContext(ctx context.Context, options ...grpc.DialOption) (c *grpc.ClientConn, err error) {
	t.m.RLock()
	c = t.conn
	t.m.RUnlock()

	if c != nil {
		if c.GetState() != connectivity.Shutdown {
			return c, nil
		} else {
			c.Close()
		}
	}

	t.m.Lock()
	defer t.m.Unlock()

	if t.conn, err = t.d.DialContext(ctx, options...); err != nil {
		return nil, err
	}

	return t.conn, nil
}

func (t *Cached) Defaults(combined ...grpc.DialOption) Defaulted {
	return t.d.Defaults(combined...)
}
