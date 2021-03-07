package clustering

import (
	"log"
	"sync"

	"github.com/james-lawrence/bw/clustering/rendezvous"

	"github.com/hashicorp/memberlist"
)

type pending interface {
	Rendezvous
	Members() []*memberlist.Node
}

// NewPending a cluster that hasn't been initialized yet but will be.
func NewPending() *Pending {
	return &Pending{
		m: &sync.Mutex{},
	}
}

// Pending set of nodes
type Pending struct {
	c pending
	m *sync.Mutex
}

func (t *Pending) get() pending {
	t.m.Lock()
	defer t.m.Unlock()
	return t.c
}

// Assign the actual cluster.
func (t *Pending) Assign(c pending) {
	t.m.Lock()
	defer t.m.Unlock()
	log.Printf("ASSIGNING: %T\n", c)
	t.c = c
}

// Members - see Cluster.
func (t *Pending) Members() []*memberlist.Node {
	c := t.get()
	if c == nil {
		return []*memberlist.Node{}
	}

	return t.c.Members()
}

// Get - see Cluster.
func (t Pending) Get(key []byte) *memberlist.Node {
	return rendezvous.Max(key, t.Members())
}

// GetN - see Cluster.
func (t Pending) GetN(n int, key []byte) []*memberlist.Node {
	return rendezvous.MaxN(n, key, t.Members())
}
