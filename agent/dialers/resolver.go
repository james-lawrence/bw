package dialers

import (
	"sync"
	"sync/atomic"

	"github.com/james-lawrence/bw/agent"
	"google.golang.org/grpc/resolver"
)

type ringResolverBuilder struct {
	c rendezvous
}

func NewRingResolver(c rendezvous) resolver.Builder {
	return &ringResolverBuilder{c: c}
}

func (b *ringResolverBuilder) Build(target resolver.Target, cc resolver.ClientConn, opts resolver.BuildOptions) (resolver.Resolver, error) {
	r := &ringResolver{
		cc: cc,
		c:  b.c,
	}
	r.update()
	return r, nil
}

// quorumScheme is the scheme of the quorum target. it must not be the scheme of a peer address
// (bw.agent://ip:port#name), a resolver registered for that scheme would resolve every direct dial to
// the members of the quorum instead of the peer being dialed.
const quorumScheme = "bw.quorum"

func (b *ringResolverBuilder) Scheme() string {
	return quorumScheme
}

type ringResolver struct {
	cc      resolver.ClientConn
	c       rendezvous
	mu      sync.Mutex
	index   atomic.Uint32
	current []resolver.Address
}

func (r *ringResolver) ResolveNow(o resolver.ResolveNowOptions) {
	r.update()
}

func (r *ringResolver) Close() {}

func (r *ringResolver) update() {
	r.mu.Lock()
	defer r.mu.Unlock()

	// shift the starting offset
	n := r.index.Add(1) % max(uint32(len(r.current)), 1)
	// 0 means we've hit every one. so we should regenerate.
	// 1 means we're just starting out.
	if n <= 1 {
		_upd := agent.QuorumPeers(r.c)
		upd := make([]resolver.Address, 0, len(_upd))

		for _, p := range _upd {
			upd = append(upd, resolver.Address{Addr: agent.RPCAddress(p)})
		}

		r.current = upd
		n = 0
		r.index.Store(1)
	}

	rotated := make([]resolver.Address, len(r.current))
	copy(rotated, r.current[n:])
	copy(rotated[len(r.current)-int(n):], r.current[:n])
	r.cc.UpdateState(resolver.State{Addresses: rotated})
}
