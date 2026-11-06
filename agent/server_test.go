package agent_test

import (
	"context"
	"io"
	"log"
	"net"
	"os"
	"strings"
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

		_, err := h.client.Deploy(context.Background(), "", &DeployOptions{}, &Archive{})
		require.NoError(t, err)
	})
}

type initiatorDeployer struct {
	initiator chan string
}

func (initiatorDeployer) Cancel()                         {}
func (initiatorDeployer) Reset() error                    { return nil }
func (initiatorDeployer) Deployments() ([]*Deploy, error) { return nil, nil }
func (initiatorDeployer) Logs([]byte) io.ReadCloser       { return io.NopCloser(strings.NewReader("")) }

func (t initiatorDeployer) Deploy(_ context.Context, by string, _ *DeployOptions, _ *Archive) (*Deploy, error) {
	t.initiator <- by
	return &Deploy{}, nil
}

func TestServerDeployInitiator(t *testing.T) {
	d := initiatorDeployer{initiator: make(chan string, 1)}

	socket, err := net.Listen("tcp", "127.0.0.1:0")
	require.NoError(t, err)
	c := cluster.New(NewPeer(fake.CharactersN(10)), clustering.NewMock(clusteringtestutil.NewNodes(1)[0]))
	grpcs := grpc.NewServer()
	RegisterAgentServer(grpcs, NewServer(c, ServerOptionAuth(testauth{}), ServerOptionDeployer(d)))
	go grpcs.Serve(socket)
	t.Cleanup(grpcs.Stop)

	conn, err := grpc.NewClient(socket.Addr().String(), grpc.WithTransportCredentials(insecure.NewCredentials()))
	require.NoError(t, err)
	t.Cleanup(func() { conn.Close() })

	_, err = NewConn(conn).Deploy(context.Background(), "test user", &DeployOptions{}, &Archive{})
	require.NoError(t, err)
	require.Equal(t, "test user", <-d.initiator)
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
