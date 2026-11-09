package dialers_test

import (
	"net/url"
	"testing"

	"github.com/hashicorp/memberlist"
	"github.com/stretchr/testify/require"
	"google.golang.org/grpc/resolver"

	"github.com/james-lawrence/bw/agent"
	. "github.com/james-lawrence/bw/agent/dialers"
	"github.com/james-lawrence/bw/clustering/clusteringtestutil"
)

type fakeRing []*memberlist.Node

func (t fakeRing) GetN(n int, key []byte) []*memberlist.Node { return t }

type fakeCC struct {
	resolver.ClientConn
	state resolver.State
}

func (t *fakeCC) UpdateState(s resolver.State) error {
	t.state = s
	return nil
}

func resolve(t *testing.T, ring fakeRing, target string) []string {
	t.Helper()

	u, err := url.Parse(target)
	require.NoError(t, err)

	cc := &fakeCC{}
	r, err := NewRingResolver(ring).Build(resolver.Target{URL: *u}, cc, resolver.BuildOptions{})
	require.NoError(t, err)
	defer r.Close()

	addrs := make([]string, 0, len(cc.state.Addresses))
	for _, a := range cc.state.Addresses {
		addrs = append(addrs, a.Addr)
	}
	return addrs
}

func TestRingResolver(t *testing.T) {
	ring := fakeRing{
		clusteringtestutil.NewNodeFromAddress("node1", "127.0.0.1"),
		clusteringtestutil.NewNodeFromAddress("node2", "127.0.0.2"),
	}

	// dialing a specific peer must reach that peer, not a member of the quorum. cancelling the deploys of a
	// cluster dials every peer, they all ended up on the same node.
	t.Run("does not resolve peer addresses", func(t *testing.T) {
		peer, err := url.Parse(agent.RPCAddress(&agent.Peer{Name: "peer", Ip: "127.0.0.9", P2PPort: 2000}))
		require.NoError(t, err)
		require.NotEqual(t, peer.Scheme, NewRingResolver(ring).Scheme())
	})

	t.Run("the quorum resolves to the members of the quorum", func(t *testing.T) {
		scheme := NewRingResolver(ring).Scheme()
		require.Len(t, resolve(t, ring, scheme+"://"), len(ring))
	})
}
