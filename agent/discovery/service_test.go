package discovery

import (
	"context"
	"fmt"
	"net"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/james-lawrence/bw/agent"
	"github.com/stretchr/testify/require"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/proto"
)

type fakecluster struct{ peers, quorum []*agent.Peer }

func (t fakecluster) Peers() []*agent.Peer  { return t.peers }
func (t fakecluster) Quorum() []*agent.Peer { return t.quorum }

func peers(n int) (r []*agent.Peer) {
	for i := 0; i < n; i++ {
		r = append(r, &agent.Peer{Name: fmt.Sprintf("n%d", i), Ip: "10.0.0.1", P2PPort: 2000, Archive: i%2 == 0, Mode: agent.PeerMode(i % 2)})
	}
	return r
}

func serve(t *testing.T, bind func(*grpc.Server)) *grpc.ClientConn {
	l, err := net.Listen("tcp", "127.0.0.1:0")
	require.NoError(t, err)
	s := grpc.NewServer()
	bind(s)
	go s.Serve(l)
	t.Cleanup(s.Stop)
	cc, err := grpc.NewClient(l.Addr().String(), grpc.WithTransportCredentials(insecure.NewCredentials()))
	require.NoError(t, err)
	t.Cleanup(func() { cc.Close() })
	return cc
}

func collect(t *testing.T, cc *grpc.ClientConn, max int64) (nodes []*Node) {
	ctx, done := context.WithTimeout(context.Background(), 5*time.Second)
	defer done()
	s, err := NewDiscoveryClient(cc).Agents(ctx, &AgentsRequest{Maximum: max})
	require.NoError(t, err)
	for {
		b, err := s.Recv()
		if err != nil {
			break
		}
		nodes = append(nodes, b.Nodes...)
	}
	return nodes
}

func TestAgentsReturnsEveryPeerPreservingFields(t *testing.T) {
	ps := peers(7)
	for _, max := range []int64{0, 1, 2, 3, 100} {
		cc := serve(t, New(fakecluster{peers: ps}).Bind)
		nodes := collect(t, cc, max)
		require.Len(t, nodes, len(ps), "maximum(%d)", max)
		for i, n := range nodes {
			require.Equal(t, ps[i].Name, n.Name)
			require.Equal(t, ps[i].Archive, n.Archive)
			require.Equal(t, int32(ps[i].Mode), n.Mode)
		}
	}
}

func TestAgentsEmpty(t *testing.T) {
	require.Empty(t, collect(t, serve(t, New(fakecluster{}).Bind), 10))
}

func TestQuorum(t *testing.T) {
	cc := serve(t, New(fakecluster{quorum: peers(3)}).Bind)
	resp, err := NewDiscoveryClient(cc).Quorum(context.Background(), &QuorumRequest{})
	require.NoError(t, err)
	require.Len(t, resp.Nodes, 3)
}

func TestPeerNodeRoundTrip(t *testing.T) {
	for _, p := range peers(2) {
		n := peerToNode(p)
		got := nodeToPeer(&n)
		require.Equal(t, p.Name, got.Name)
		require.Equal(t, p.Ip, got.Ip)
		require.Equal(t, p.P2PPort, got.P2PPort)
		require.Equal(t, p.Archive, got.Archive)
		require.Equal(t, p.Mode, got.Mode)
	}
}

func TestAuthorityCheck(t *testing.T) {
	path := filepath.Join(t.TempDir(), "authority.proto")
	encoded, err := proto.Marshal(&agent.TLSCertificates{Fingerprint: "abc"})
	require.NoError(t, err)
	require.NoError(t, os.WriteFile(path, encoded, 0600))

	check := func(p, fp string) codes.Code {
		cc := serve(t, NewAuthority(p).Bind)
		_, err := NewAuthorityClient(cc).Check(context.Background(), &CheckRequest{Fingerprint: fp})
		return status.Code(err)
	}

	require.Equal(t, codes.OK, check(path, "abc"))
	require.Equal(t, codes.NotFound, check(path, "nope"))
	require.Equal(t, codes.Unavailable, check(filepath.Join(t.TempDir(), "missing"), "abc"))

	bad := filepath.Join(t.TempDir(), "bad")
	require.NoError(t, os.WriteFile(bad, []byte{0xff, 0xff, 0xff}, 0600))
	require.Equal(t, codes.Unavailable, check(bad, "abc"))
}
