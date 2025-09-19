package clustering

import (
	"context"
	"sync"
	"time"

	"github.com/hashicorp/memberlist"
)

type cachefiller func(context.Context) Rendezvous

func NewCached(fetch cachefiller) *Cached {
	return &Cached{
		fetch: fetch,
		ttl:   time.Second,
	}
}

type Cached struct {
	m       sync.RWMutex
	fetch   cachefiller
	ttl     time.Duration
	last    time.Time
	_cached Rendezvous
}

func (t *Cached) cached() Rendezvous {
	t.m.RLock()
	if time.Since(t.last) <= t.ttl && t._cached != nil {
		cached := t._cached
		t.m.RUnlock()
		return cached
	}
	t.m.RUnlock()

	t.m.Lock()
	defer t.m.Unlock()

	if time.Since(t.last) <= t.ttl && t._cached != nil {
		return t._cached
	}

	t._cached = t.fetch(context.Background())
	t.last = time.Now()
	return t._cached
}

func (t *Cached) Members() []*memberlist.Node {
	return t.cached().Members()
}

func (t *Cached) Get(key []byte) *memberlist.Node {
	return t.cached().Get(key)
}

func (t *Cached) GetN(n int, key []byte) []*memberlist.Node {
	return t.cached().GetN(n, key)
}
