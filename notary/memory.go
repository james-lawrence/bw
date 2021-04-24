package notary

import (
	"context"
	"errors"
	"sync"
)

func NewMem(grants ...*Grant) memory {
	m := make(map[string]*Grant, len(grants))
	for _, g := range grants {
		m[g.Fingerprint] = g
	}

	return memory{
		mem: m,
		m:   &sync.RWMutex{},
	}
}

// memory storage
type memory struct {
	mem map[string]*Grant
	m   *sync.RWMutex
}

func (t memory) UnsafeSnapshot() (dst []*Grant) {
	dst = make([]*Grant, 0, len(t.mem))
	for _, v := range t.mem {
		dst = append(dst, v)
	}

	return dst
}

func (t memory) Lookup(fingerprint string) (g *Grant, err error) {
	var (
		ok bool
	)

	t.m.RLock()
	defer t.m.RUnlock()

	if g, ok = t.mem[fingerprint]; !ok {
		return g, errors.New("fingerprint not found")
	}

	return g, nil
}

func (t memory) Insert(g *Grant) (_ *Grant, err error) {
	t.m.Lock()
	defer t.m.Unlock()

	gd := g.EnsureDefaults()
	t.mem[g.Fingerprint] = gd

	return gd, nil
}

func (t memory) Delete(g *Grant) (r *Grant, err error) {
	t.m.Lock()
	defer t.m.Unlock()

	if r, ok := t.mem[g.Fingerprint]; ok {
		g = r
	}

	delete(t.mem, g.Fingerprint)

	return g, nil
}

func (t memory) Sync(ctx context.Context, b Bloomy, c chan *Grant) (err error) {
	t.m.RLock()
	defer t.m.RUnlock()

	for k, g := range t.mem {
		if b.Test([]byte(k)) {
			continue
		}

		select {
		case c <- g:
		case <-ctx.Done():
			return ctx.Err()
		}
	}

	return nil
}
