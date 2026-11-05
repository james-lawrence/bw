package bootstrap_test

import (
	"context"
	"io"
	"log"
	"os"
	"testing"
	"time"

	"github.com/pkg/errors"
	"github.com/stretchr/testify/require"

	"github.com/james-lawrence/bw"
	"github.com/james-lawrence/bw/agent"
	"github.com/james-lawrence/bw/agenttestutil"
	. "github.com/james-lawrence/bw/bootstrap"
	"github.com/james-lawrence/bw/deployment"
	"github.com/james-lawrence/bw/internal/testingx"
	"github.com/james-lawrence/bw/storage"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

func TestBootstrap(t *testing.T) {
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
		missing = status.Error(codes.NotFound, "missing deployment")
	)

	t.Run("should succeed when no errors occur", func(t *testing.T) {
		c := agent.Config{
			Root: testingx.TempDir(t),
		}
		current := &agent.Deploy{
			Stage:   agent.Deploy_Completed,
			Archive: &archive1,
			Options: &dopts1,
		}
		reg := storage.NoopRegistry{}
		p := agent.NewPeer("local")

		dc := deployment.New(
			p,
			noopDeployer{err: nil},
			deployment.CoordinatorOptionStorage(reg),
			deployment.CoordinatorOptionRoot(testingx.TempDir(t)),
		)
		require.NoError(t, Run(context.Background(), SocketLocal(c), &agenttestutil.Mock{Fail: missing}))
		require.NoError(t, Run(context.Background(), SocketQuorum(c), &agenttestutil.Mock{Current: current}))
		require.NoError(t, Bootstrap(context.Background(), c, dc, nil))
	})

	t.Run("should fail when it fails to download the archive", func(t *testing.T) {
		reg := storage.NoopRegistry{Err: errors.New("download failed")}
		p := agent.NewPeer("local")

		c := agent.Config{
			Root: testingx.TempDir(t),
		}
		current := &agent.Deploy{
			Stage:   agent.Deploy_Completed,
			Archive: &archive1,
			Options: &dopts1,
		}

		dc := deployment.New(
			p,
			noopDeployer{err: nil},
			deployment.CoordinatorOptionStorage(reg),
			deployment.CoordinatorOptionRoot(testingx.TempDir(t)),
		)
		require.NoError(t, Run(context.Background(), SocketLocal(c), &agenttestutil.Mock{Fail: missing}))
		require.NoError(t, Run(context.Background(), SocketQuorum(c), &agenttestutil.Mock{Current: current}))
		require.EqualError(t, errors.Cause(Bootstrap(context.Background(), c, dc, nil)), "download failed")
	})

	t.Run("should fail when the deployment fails", func(t *testing.T) {
		c := agent.Config{
			Root: testingx.TempDir(t),
		}
		current := &agent.Deploy{
			Stage:   agent.Deploy_Completed,
			Archive: &archive1,
			Options: &dopts1,
		}
		reg := storage.NoopRegistry{}
		p := agent.NewPeer("local")

		dc := deployment.New(
			p,
			noopDeployer{err: errors.New("deployment failed")},
			deployment.CoordinatorOptionStorage(reg),
			deployment.CoordinatorOptionRoot(testingx.TempDir(t)),
		)
		require.NoError(t, Run(context.Background(), SocketLocal(c), &agenttestutil.Mock{Fail: missing}))
		require.NoError(t, Run(context.Background(), SocketQuorum(c), &agenttestutil.Mock{Current: current}))
		require.EqualError(t, errors.Cause(Bootstrap(context.Background(), c, dc, nil)), "deployment failed")
	})

	t.Run("should succeed when it finishes bootstrapping from quorum", func(t *testing.T) {
		c := agent.Config{
			Root: testingx.TempDir(t),
		}
		current := &agent.Deploy{
			Stage:   agent.Deploy_Completed,
			Archive: &archive1,
			Options: &dopts1,
		}
		reg := storage.NoopRegistry{}
		p := agent.NewPeer("local")

		dc := deployment.New(
			p,
			noopDeployer{err: nil},
			deployment.CoordinatorOptionStorage(reg),
			deployment.CoordinatorOptionRoot(testingx.TempDir(t)),
		)
		require.NoError(t, Run(context.Background(), SocketLocal(c), &agenttestutil.Mock{Fail: missing}))
		require.NoError(t, Run(context.Background(), SocketQuorum(c), &agenttestutil.Mock{Current: current}))
		require.NoError(t, Bootstrap(context.Background(), c, dc, nil))
	})

	t.Run("should bootstrap from fallback bootstrap services when quorum has no deployments", func(t *testing.T) {
		c := agent.Config{
			Root: testingx.TempDir(t),
		}
		current := &agent.Deploy{
			Stage:   agent.Deploy_Completed,
			Archive: &archive1,
			Options: &dopts1,
		}
		reg := storage.NoopRegistry{}
		p := agent.NewPeer("local")

		dc := deployment.New(
			p,
			noopDeployer{err: nil},
			deployment.CoordinatorOptionStorage(reg),
			deployment.CoordinatorOptionRoot(testingx.TempDir(t)),
		)
		require.NoError(t, Run(context.Background(), SocketLocal(c), &agenttestutil.Mock{Fail: missing}))
		require.NoError(t, Run(context.Background(), SocketQuorum(c), &agenttestutil.Mock{Fail: missing}))
		require.NoError(t, Run(context.Background(), SocketAuto(c), &agenttestutil.Mock{Current: current}))
		results := make(chan *deployment.DeployResult, 1)
		require.EqualError(t, Bootstrap(context.Background(), c, dc, results), "failed to determine latest deployment from quorum, retrying: no deployments found")
		result := <-results
		require.Equal(t, current.Archive.DeploymentID, []byte(result.ID))
	})

	t.Run("should stop attempting to bootstrap if all services return no deployments found", func(t *testing.T) {
		c := agent.Config{
			Root: testingx.TempDir(t),
		}

		reg := storage.NoopRegistry{}
		p := agent.NewPeer("local")

		dc := deployment.New(
			p,
			noopDeployer{err: nil},
			deployment.CoordinatorOptionStorage(reg),
			deployment.CoordinatorOptionRoot(testingx.TempDir(t)),
		)
		require.NoError(t, Run(context.Background(), SocketLocal(c), &agenttestutil.Mock{Fail: missing}))
		require.NoError(t, Run(context.Background(), SocketQuorum(c), &agenttestutil.Mock{Fail: missing}))
		require.NoError(t, Run(context.Background(), SocketAuto(c), &agenttestutil.Mock{Fail: missing}))
		require.NoError(t, Bootstrap(context.Background(), c, dc, nil))
	})

	t.Run("active deploy", func(t *testing.T) {
		t.Run("should deploy an active deploy from quorum", func(t *testing.T) {
			c := agent.Config{
				Root: testingx.TempDir(t),
			}
			current := &agent.Deploy{
				Stage:   agent.Deploy_Completed,
				Archive: &archive1,
				Options: &dopts1,
			}
			reg := storage.NoopRegistry{}
			p := agent.NewPeer("local")

			dc := deployment.New(
				p,
				noopDeployer{err: nil},
				deployment.CoordinatorOptionStorage(reg),
				deployment.CoordinatorOptionRoot(testingx.TempDir(t)),
			)
			require.NoError(t, Run(context.Background(), SocketLocal(c), &agenttestutil.Mock{Current: current}))
			require.NoError(t, Run(context.Background(), SocketQuorum(c), &agenttestutil.Mock{Current: current, Info: agent.ArchiveResponse_ActiveDeploy}))
			require.Equal(t, "active deploy matches the local deployment, waiting for deployment to complete: deployment in progress", Bootstrap(context.Background(), c, dc, nil).Error())
		})
	})
}

func TestMain(m *testing.M) {
	log.SetOutput(io.Discard)
	code := m.Run()
	testingx.Cleanup()
	os.Exit(code)
}
