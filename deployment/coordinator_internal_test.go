package deployment

import (
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"github.com/james-lawrence/bw"
	"github.com/james-lawrence/bw/agent"
	"github.com/james-lawrence/bw/internal/testingx"
	"github.com/james-lawrence/bw/storage"
)

func TestCoordinator(t *testing.T) {

	t.Run("should autocorrect dead deploys", func(t *testing.T) {
		workdir := testingx.TempDir(t)
		id := bw.MustGenerateID()
		deploydir := filepath.Join(workdir, "deploys", id.String())

		require.NoError(t, os.MkdirAll(deploydir, 0755))

		p := agent.NewPeer("node1")
		c := New(
			p,
			NewDirective(),
			CoordinatorOptionRoot(workdir),
			CoordinatorOptionStorage(storage.NoopRegistry{}),
		)

		deploys, err := c.Deployments()
		require.NoError(t, err)
		require.Len(t, deploys, 0)
		a := agent.Archive{
			DeploymentID: id,
			Peer:         p,
		}
		dopts := agent.DeployOptions{
			Concurrency:    1,
			IgnoreFailures: false,
			Timeout:        int64(time.Minute),
		}

		require.NoError(t, writeDeployMetadata(deploydir, &agent.Deploy{
			Initiator: "test user",
			Archive:   &a,
			Options:   &dopts,
			Stage:     agent.Deploy_Deploying,
		}))

		deploys, err = c.Deployments()
		require.NoError(t, err)
		require.Len(t, deploys, 1)
		require.Equal(t, agent.Deploy_Failed, deploys[0].Stage)
	})

	t.Run("Reset should properly reset deploys directory", func(t *testing.T) {
		check := func(t *testing.T, s agent.Deploy_Stage, result int) {
			workdir := testingx.TempDir(t)
			p := agent.NewPeer("node1")
			c := New(
				p,
				NewDirective(),
				CoordinatorOptionRoot(workdir),
				CoordinatorOptionStorage(storage.NoopRegistry{}),
			)

			deploys, err := c.Deployments()
			require.NoError(t, err)
			require.Len(t, deploys, 0)

			dopts := agent.DeployOptions{
				Concurrency:       1,
				IgnoreFailures:    false,
				SilenceDeployLogs: true,
				Timeout:           int64(time.Minute),
			}
			a := agent.Archive{
				DeploymentID: bw.MustGenerateID(),
				Peer:         p,
			}
			deploydir := filepath.Join(workdir, "deploys", bw.RandomID(a.DeploymentID).String())
			require.NoError(t, os.MkdirAll(deploydir, 0755))
			require.NoError(t, writeDeployMetadata(deploydir, &agent.Deploy{
				Initiator: "test user",
				Archive:   &a,
				Options:   &dopts,
				Stage:     s,
			}))
			deploys, err = c.Deployments()
			require.NoError(t, err)
			require.Len(t, deploys, 1)

			require.NoError(t, c.Reset())

			deploys, err = c.Deployments()
			require.NoError(t, err)
			require.Len(t, deploys, result)
		}
		t.Run("failed deploy should be removed", func(t *testing.T) { check(t, agent.Deploy_Failed, 0) })
		t.Run("currently deploying should be removed", func(t *testing.T) { check(t, agent.Deploy_Deploying, 0) })
		t.Run("completed deploy should remain", func(t *testing.T) { check(t, agent.Deploy_Completed, 1) })
	})
}
