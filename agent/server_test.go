package agent_test

import (
	"context"
	"io"
	"log"
	"net"
	"os"
	"testing"

	. "github.com/james-lawrence/bw/agent"
	"github.com/james-lawrence/bw/agent/dialers"
	"github.com/james-lawrence/bw/cluster"
	"github.com/james-lawrence/bw/clustering"
	"github.com/james-lawrence/bw/clustering/clusteringtestutil"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/protobuf/proto"

	"github.com/icrowley/fake"
	"github.com/stretchr/testify/require"
)

type testauth struct{}

func (t testauth) Deploy(ctx context.Context) error {
	return nil
}

func TestMain(m *testing.M) {
	log.SetOutput(io.Discard)
	os.Exit(m.Run())
}

type harness struct {
	client   Client
	cluster  cluster.Cluster
	listener net.Listener
}

func testClient(t *testing.T) harness {
	t.Helper()

	socket, err := net.Listen("tcp", ":0")
	require.NoError(t, err)
	peers := clusteringtestutil.NewNodes(5)

	c := cluster.New(
		NewPeer(fake.CharactersN(10)),
		clustering.NewMock(peers[0], peers[1:]...),
	)
	s := NewServer(c, ServerOptionAuth(testauth{}))

	grpcs := grpc.NewServer()
	RegisterAgentServer(grpcs, s)
	go func() {
		grpcs.Serve(socket)
	}()

	conn, err := dialers.NewDirect(
		socket.Addr().String(),
		grpc.WithTransportCredentials(insecure.NewCredentials()),
	).DialContext(context.Background())
	require.NoError(t, err)

	t.Cleanup(func() {
		require.NoError(t, socket.Close())
	})

	return harness{client: NewConn(conn), cluster: c, listener: socket}
}

func TestServerConnect(t *testing.T) {
	t.Run("should return cluster details", func(t *testing.T) {
		h := testClient(t)

		q := h.cluster.Quorum()
		info, err := h.client.Connect(context.Background())
		require.NoError(t, err)
		require.ElementsMatch(t, []*Peer{q[0], q[1], q[2]}, info.Quorum)
	})
}

func TestServerInfo(t *testing.T) {
	t.Run("should return information about the agent", func(t *testing.T) {
		h := testClient(t)

		info, err := h.client.Info(context.Background())
		require.NoError(t, err)
		require.Empty(t, info.Deployments)
		tmp := h.cluster.Local()
		tmp.Status = Peer_Node
		require.True(t, proto.Equal(info.Peer, tmp))
	})
}

func TestServerDeploy(t *testing.T) {
	t.Run("should trigger a deploy on the server", func(t *testing.T) {
		h := testClient(t)

		_, err := h.client.Deploy(context.Background(), &DeployOptions{}, &Archive{})
		require.NoError(t, err)
	})
}

func TestServerLogs(t *testing.T) {
	t.Run("should return logs from the given deployer", func(t *testing.T) {
		h := testClient(t)
		p := h.cluster.Local()
		pipe := h.client.Logs(context.Background(), p, []byte("fake"))
		raw, err := io.ReadAll(pipe)
		require.NoError(t, err)
		require.Equal(t, "INFO: fake", string(raw))
	})
}
