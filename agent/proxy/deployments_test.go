package proxy_test

import (
	"context"
	"io"
	"net"
	"sync/atomic"
	"testing"

	"github.com/hashicorp/memberlist"
	"github.com/james-lawrence/bw/agent"
	"github.com/james-lawrence/bw/agent/dialers"
	"github.com/james-lawrence/bw/agent/proxy"
	"github.com/james-lawrence/bw/notary"
	"github.com/stretchr/testify/require"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/status"
)

type perm struct{ deploy bool }

func (t perm) Authorize(ctx context.Context) *notary.Permission {
	return &notary.Permission{Deploy: t.deploy}
}

type ring struct{ nodes []*memberlist.Node }

func (t ring) GetN(n int, key []byte) []*memberlist.Node { return t.nodes }

// fakeQuorum records what reached the quorum.
type fakeQuorum struct {
	agent.UnimplementedQuorumServer
	deploys, cancels, uploads atomic.Int64
	chunks                    atomic.Int64
}

func (t *fakeQuorum) Deploy(ctx context.Context, r *agent.DeployCommandRequest) (*agent.DeployCommandResult, error) {
	t.deploys.Add(1)
	return &agent.DeployCommandResult{}, nil
}

func (t *fakeQuorum) Cancel(ctx context.Context, r *agent.CancelRequest) (*agent.CancelResponse, error) {
	t.cancels.Add(1)
	return &agent.CancelResponse{}, nil
}

func (t *fakeQuorum) Upload(s agent.Quorum_UploadServer) error {
	t.uploads.Add(1)
	for {
		if _, err := s.Recv(); err != nil {
			if err == io.EOF {
				return s.SendAndClose(&agent.UploadResponse{})
			}
			return err
		}
		t.chunks.Add(1)
	}
}

func listen(t *testing.T, bind func(*grpc.Server)) net.Listener {
	l, err := net.Listen("tcp", "127.0.0.1:0")
	require.NoError(t, err)
	s := grpc.NewServer()
	bind(s)
	go s.Serve(l)
	t.Cleanup(s.Stop)
	return l
}

// client connects to a proxy whose quorum dialer reaches the fake quorum.
func client(t *testing.T, auth perm, q *fakeQuorum) agent.DeploymentsClient {
	ql := listen(t, func(s *grpc.Server) { agent.RegisterQuorumServer(s, q) })

	node := agent.PeerToNode(&agent.Peer{Name: "q1", Ip: "127.0.0.1", P2PPort: 1})
	d := dialers.NewQuorum(
		ring{nodes: []*memberlist.Node{node}},
		grpc.WithTransportCredentials(insecure.NewCredentials()),
		grpc.WithContextDialer(func(ctx context.Context, _ string) (net.Conn, error) {
			return (&net.Dialer{}).DialContext(ctx, "tcp", ql.Addr().String())
		}),
	)

	pl := listen(t, proxy.NewDeployment(auth, d).Bind)
	cc, err := grpc.NewClient(pl.Addr().String(), grpc.WithTransportCredentials(insecure.NewCredentials()))
	require.NoError(t, err)
	t.Cleanup(func() { cc.Close() })
	return agent.NewDeploymentsClient(cc)
}

func TestForwardsAuthorizedRequests(t *testing.T) {
	q := &fakeQuorum{}
	c := client(t, perm{deploy: true}, q)
	ctx := context.Background()

	_, err := c.Deploy(ctx, &agent.DeployCommandRequest{})
	require.NoError(t, err)
	_, err = c.Cancel(ctx, &agent.CancelRequest{})
	require.NoError(t, err)

	up, err := c.Upload(ctx)
	require.NoError(t, err)
	for i := 0; i < 3; i++ {
		require.NoError(t, up.Send(&agent.UploadChunk{}))
	}
	_, err = up.CloseAndRecv()
	require.NoError(t, err)

	require.EqualValues(t, 1, q.deploys.Load())
	require.EqualValues(t, 1, q.cancels.Load())
	require.EqualValues(t, 1, q.uploads.Load())
	require.EqualValues(t, 3, q.chunks.Load())
}

// without the deploy permission nothing may reach the quorum.
func TestDeniesUnauthorizedRequests(t *testing.T) {
	q := &fakeQuorum{}
	c := client(t, perm{deploy: false}, q)
	ctx := context.Background()

	_, err := c.Deploy(ctx, &agent.DeployCommandRequest{})
	require.Equal(t, codes.PermissionDenied, status.Code(err))
	_, err = c.Cancel(ctx, &agent.CancelRequest{})
	require.Equal(t, codes.PermissionDenied, status.Code(err))

	up, err := c.Upload(ctx)
	require.NoError(t, err)
	up.Send(&agent.UploadChunk{})
	_, err = up.CloseAndRecv()
	require.Equal(t, codes.PermissionDenied, status.Code(err))

	w, err := c.Watch(ctx, &agent.WatchRequest{})
	require.NoError(t, err)
	_, err = w.Recv()
	require.Equal(t, codes.PermissionDenied, status.Code(err))

	l, err := c.Logs(ctx, &agent.LogRequest{Peer: &agent.Peer{}})
	require.NoError(t, err)
	_, err = l.Recv()
	require.Equal(t, codes.PermissionDenied, status.Code(err))

	require.Zero(t, q.deploys.Load()+q.cancels.Load()+q.uploads.Load())
}

func TestLogsRequiresPeer(t *testing.T) {
	c := client(t, perm{deploy: true}, &fakeQuorum{})
	l, err := c.Logs(context.Background(), &agent.LogRequest{})
	require.NoError(t, err)
	_, err = l.Recv()
	require.Equal(t, codes.FailedPrecondition, status.Code(err))
}
