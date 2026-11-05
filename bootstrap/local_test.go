package bootstrap_test

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"github.com/james-lawrence/bw"
	"github.com/james-lawrence/bw/agent"
	"github.com/james-lawrence/bw/agenttestutil"
	"github.com/james-lawrence/bw/agentutil"
	. "github.com/james-lawrence/bw/bootstrap"
	"github.com/james-lawrence/bw/internal/testingx"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

func TestLocal(t *testing.T) {
	var (
		peer1    = agent.NewPeer("node1")
		archive1 = agent.Archive{
			Peer:         peer1,
			Ts:           time.Now().Unix(),
			DeploymentID: bw.MustGenerateID(),
		}
		dopts1 = agent.DeployOptions{
			Timeout:           int64(time.Hour),
			SilenceDeployLogs: true,
		}
	)

	t.Run("should succeed when no errors occur", func(t *testing.T) {
		c := agent.Config{
			Root: testingx.TempDir(t),
		}
		p := agent.NewPeer("local")

		d, srv := testingx.NewGRPCServer2(t, func(srv *grpc.Server) {
			(&agenttestutil.FakeAgent{
				StatusResponse: agent.StatusResponse{
					Deployments: []*agent.Deploy{
						{
							Stage:   agent.Deploy_Completed,
							Archive: &archive1,
							Options: &dopts1,
						},
					},
				},
			}).Bind(srv)
		})
		defer testingx.GRPCCleanup(t, nil, srv)

		require.NoError(t, Run(context.Background(), SocketQuorum(c), NewLocal(p, d)))
		_, err := Latest(context.Background(), SocketQuorum(c), grpc.WithTransportCredentials(insecure.NewCredentials()))
		require.NoError(t, err)
	})

	t.Run("should return no deployments error when no deployments exist", func(t *testing.T) {
		c := agent.Config{
			Root: testingx.TempDir(t),
		}
		p := agent.NewPeer("local")

		d, srv := testingx.NewGRPCServer2(t, func(srv *grpc.Server) {
			(&agenttestutil.FakeAgent{
				StatusResponse: agent.StatusResponse{
					Deployments: []*agent.Deploy{},
				},
			}).Bind(srv)
		})
		defer testingx.GRPCCleanup(t, nil, srv)

		require.NoError(t, Run(context.Background(), SocketQuorum(c), NewLocal(p, d)))
		_, err := Latest(context.Background(), SocketQuorum(c), grpc.WithTransportCredentials(insecure.NewCredentials()))
		require.Equal(t, agentutil.ErrNoDeployments, err)
	})

	t.Run("should error out when an error occurrs", func(t *testing.T) {
		c := agent.Config{
			Root: testingx.TempDir(t),
		}
		p := agent.NewPeer("local")

		d, srv := testingx.NewGRPCServer2(t, func(srv *grpc.Server) {
			(&agenttestutil.FakeAgent{
				ErrResult: errors.New("boom"),
			}).Bind(srv)
		})
		defer testingx.GRPCCleanup(t, nil, srv)

		require.NoError(t, Run(context.Background(), SocketQuorum(c), NewLocal(p, d)))
		_, err := Latest(context.Background(), SocketQuorum(c), grpc.WithTransportCredentials(insecure.NewCredentials()))
		require.Error(t, err)
	})
}
