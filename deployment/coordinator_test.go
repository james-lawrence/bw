package deployment_test

import (
	"context"
	"fmt"
	"io"
	"log"
	"os"
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"github.com/james-lawrence/bw"
	"github.com/james-lawrence/bw/agent"
	"github.com/james-lawrence/bw/deployment"
	"github.com/james-lawrence/bw/internal/testingx"
	"github.com/james-lawrence/bw/storage"
)

func TestCoordinator(t *testing.T) {

	t.Run("deployments should return the deploy options", func(t *testing.T) {
		workdir := testingx.TempDir(t)
		p := agent.NewPeer("node1")
		c := deployment.New(
			p,
			deployment.NewDirective(),
			deployment.CoordinatorOptionRoot(workdir),
			deployment.CoordinatorOptionStorage(storage.NoopRegistry{}),
		)
		deploys, err := c.Deployments()
		require.NoError(t, err)
		require.Len(t, deploys, 0)
		a := &agent.Archive{
			DeploymentID: bw.MustGenerateID(),
			Peer:         p,
		}
		dopts := &agent.DeployOptions{
			Concurrency:       1,
			IgnoreFailures:    true,
			SilenceDeployLogs: true,
			Timeout:           int64(time.Minute),
		}

		_, err = c.Deploy(context.Background(), "test user", dopts, a)
		require.NoError(t, err)

		deploys, err = c.Deployments()
		require.NoError(t, err)
		require.Len(t, deploys, 1)
		require.NotNil(t, deploys[0].Options)
		opts := deploys[0].Options
		require.Equal(t, int64(time.Minute), opts.Timeout)
		require.Equal(t, true, opts.IgnoreFailures)
	})

	t.Run("should prevent deploys if one is already running", func(t *testing.T) {
		workdir := testingx.TempDir(t)
		t.Skip("pending")
		p := agent.NewPeer("node1")
		c := deployment.New(
			p,
			deployment.NewDirective(),
			deployment.CoordinatorOptionRoot(workdir),
			deployment.CoordinatorOptionStorage(storage.NoopRegistry{}),
		)
		deploys, err := c.Deployments()
		require.NoError(t, err)
		require.Len(t, deploys, 0)
		a := &agent.Archive{
			DeploymentID: bw.MustGenerateID(),
			Peer:         p,
		}
		dopts := &agent.DeployOptions{
			Concurrency:       1,
			IgnoreFailures:    false,
			SilenceDeployLogs: true,
			Timeout:           int64(time.Minute),
		}

		_, err = c.Deploy(context.Background(), "test user", dopts, a)
		require.NoError(t, err)

		deploys, err = c.Deployments()
		require.NoError(t, err)
		require.Len(t, deploys, 1)

		a2 := &agent.Archive{
			DeploymentID: bw.MustGenerateID(),
			Peer:         p,
		}
		_, err = c.Deploy(context.Background(), "test user 2", dopts, a2)
		require.EqualError(t, err, fmt.Sprintf("test user is already deploying: %s - Deploying", bw.RandomID(a.DeploymentID).String()))

		require.Eventually(t, func() bool {
			deploys, err := c.Deployments()
			require.NoError(t, err)
			return len(deploys) == 2
		}, time.Second, 10*time.Millisecond)
	})

	t.Run("should be able to cancel a running deploy", func(t *testing.T) {
		workdir := testingx.TempDir(t)
		p := agent.NewPeer("node1")
		c := deployment.New(
			p,
			deployment.NewDirective(),
			deployment.CoordinatorOptionRoot(workdir),
			deployment.CoordinatorOptionStorage(storage.NoopRegistry{}),
		)
		deploys, err := c.Deployments()
		require.NoError(t, err)
		require.Len(t, deploys, 0)
		a := &agent.Archive{
			DeploymentID: bw.MustGenerateID(),
			Peer:         p,
		}
		dopts := &agent.DeployOptions{
			Concurrency:       1,
			IgnoreFailures:    false,
			SilenceDeployLogs: true,
			Timeout:           int64(time.Minute),
		}

		_, err = c.Deploy(context.Background(), "test user", dopts, a)
		require.NoError(t, err)

		c.Cancel()

		a2 := &agent.Archive{
			DeploymentID: bw.MustGenerateID(),
			Peer:         p,
		}
		_, err = c.Deploy(context.Background(), "test user 2", dopts, a2)
		require.NoError(t, err)

		deploys, err = c.Deployments()
		require.NoError(t, err)
		require.Len(t, deploys, 2)
	})

	t.Run("should be safe to cancel a deploy multiple times", func(t *testing.T) {
		workdir := testingx.TempDir(t)
		p := agent.NewPeer("node1")
		c := deployment.New(
			p,
			deployment.NewDirective(),
			deployment.CoordinatorOptionRoot(workdir),
			deployment.CoordinatorOptionStorage(storage.NoopRegistry{}),
		)
		deploys, err := c.Deployments()
		require.NoError(t, err)
		require.Len(t, deploys, 0)
		a := &agent.Archive{
			DeploymentID: bw.MustGenerateID(),
			Peer:         p,
		}
		dopts := &agent.DeployOptions{
			Concurrency:       1,
			IgnoreFailures:    false,
			SilenceDeployLogs: true,
			Timeout:           int64(time.Minute),
		}

		_, err = c.Deploy(context.Background(), "test user", dopts, a)
		require.NoError(t, err)

		c.Cancel()
		c.Cancel()
	})
}

func TestMain(m *testing.M) {
	log.SetFlags(log.Flags() | log.Lshortfile)
	log.SetOutput(io.Discard)
	code := m.Run()
	testingx.Cleanup()
	os.Exit(code)
}
