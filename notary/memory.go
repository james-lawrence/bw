package notary

import (
	"errors"
	"sync"
)

func newMem() memory {
	return memory{
		mem: map[string]Grant{},
		m:   &sync.RWMutex{},
	}
}

// memory storage
type memory struct {
	mem map[string]Grant
	m   *sync.RWMutex
}

func (t memory) Lookup(fingerprint string) (g Grant, err error) {
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

func (t memory) Insert(g Grant) (_ Grant, err error) {
	t.m.Lock()
	defer t.m.Unlock()

	g = g.EnsureDefaults()
	t.mem[g.Fingerprint] = g

	return g, nil
}

func (t memory) Delete(g Grant) (r Grant, err error) {
	t.m.Lock()
	defer t.m.Unlock()

	if r, ok := t.mem[g.Fingerprint]; ok {
		g = r
	}

	delete(t.mem, g.Fingerprint)

	return g, nil
}
