package deployment_test

import (
	"context"
	"fmt"
	"io"
	"log"
	"os"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"github.com/james-lawrence/bw"
	"github.com/james-lawrence/bw/agent"
	"github.com/james-lawrence/bw/deployment"
	"github.com/james-lawrence/bw/internal/testingx"
	"github.com/james-lawrence/bw/storage"
)

// blockingDeployer keeps every deploy running until the test ends.
type blockingDeployer struct{ release chan struct{} }

func (t blockingDeployer) Deploy(dctx *deployment.DeployContext) {
	go func() {
		<-t.release
		dctx.Done(nil)
	}()
}

func blocking(t *testing.T) blockingDeployer {
	d := blockingDeployer{release: make(chan struct{})}
	t.Cleanup(func() { close(d.release) })
	return d
}

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

	t.Run("redelivering the running deploy is not a conflict", func(t *testing.T) {
		workdir := testingx.TempDir(t)
		p := agent.NewPeer("node1")
		c := deployment.New(
			p,
			deployment.NewDirective(),
			deployment.CoordinatorOptionRoot(workdir),
			deployment.CoordinatorOptionStorage(storage.NoopRegistry{}),
		)
		a := &agent.Archive{DeploymentID: bw.MustGenerateID(), Peer: p}
		dopts := &agent.DeployOptions{Concurrency: 1, SilenceDeployLogs: true, Timeout: int64(time.Minute)}

		first, err := c.Deploy(context.Background(), "test user", dopts, a)
		require.NoError(t, err)

		second, err := c.Deploy(context.Background(), "", dopts, a)
		require.NoError(t, err)
		require.Equal(t, first.Archive.DeploymentID, second.Archive.DeploymentID)
	})

	t.Run("a rejected deploy must not disturb the running deploy", func(t *testing.T) {
		workdir := testingx.TempDir(t)
		p := agent.NewPeer("node1")
		c := deployment.New(
			p,
			blocking(t),
			deployment.CoordinatorOptionRoot(workdir),
			deployment.CoordinatorOptionStorage(storage.NoopRegistry{}),
		)
		dopts := &agent.DeployOptions{Concurrency: 1, SilenceDeployLogs: true, Timeout: int64(time.Minute)}
		a := &agent.Archive{DeploymentID: bw.MustGenerateID(), Peer: p}
		conflicting := &agent.Archive{DeploymentID: bw.MustGenerateID(), Peer: p}

		_, err := c.Deploy(context.Background(), "test user", dopts, a)
		require.NoError(t, err)

		_, err = c.Deploy(context.Background(), "test user 2", dopts, conflicting)
		require.EqualError(t, err, fmt.Sprintf("test user is already deploying: %s - Deploying", bw.RandomID(a.DeploymentID).String()))

		// give a rejected deploy's background work the chance to run.
		time.Sleep(100 * time.Millisecond)

		// the rejected deploy leaves no record and the running deploy is untouched.
		deploys, err := c.Deployments()
		require.NoError(t, err)
		require.Len(t, deploys, 1)
		require.Equal(t, a.DeploymentID, deploys[0].Archive.DeploymentID)
		require.Equal(t, agent.Deploy_Deploying, deploys[0].Stage)

		// still running: a conflicting deploy is rejected again and the running deploy can be redelivered.
		_, err = c.Deploy(context.Background(), "test user 2", dopts, conflicting)
		require.ErrorContains(t, err, "is already deploying")

		current, err := c.Deploy(context.Background(), "", dopts, a)
		require.NoError(t, err)
		require.Equal(t, agent.Deploy_Deploying, current.Stage)
		require.Equal(t, a.DeploymentID, current.Archive.DeploymentID)
	})

	t.Run("concurrently delivering the running deploy is not a conflict", func(t *testing.T) {
		workdir := testingx.TempDir(t)
		p := agent.NewPeer("node1")
		c := deployment.New(
			p,
			blocking(t),
			deployment.CoordinatorOptionRoot(workdir),
			deployment.CoordinatorOptionStorage(storage.NoopRegistry{}),
		)
		dopts := &agent.DeployOptions{Concurrency: 1, SilenceDeployLogs: true, Timeout: int64(time.Minute)}
		a := &agent.Archive{DeploymentID: bw.MustGenerateID(), Peer: p}

		const n = 16
		var (
			wg   sync.WaitGroup
			errs = make([]error, n)
		)

		for i := 0; i < n; i++ {
			wg.Add(1)
			go func() {
				defer wg.Done()
				_, errs[i] = c.Deploy(context.Background(), "test user", dopts, a)
			}()
		}
		wg.Wait()

		for _, err := range errs {
			require.NoError(t, err)
		}

		time.Sleep(100 * time.Millisecond)

		deploys, err := c.Deployments()
		require.NoError(t, err)
		require.Len(t, deploys, 1)
		require.Equal(t, agent.Deploy_Deploying, deploys[0].Stage)

		// the deploy is still the running one.
		_, err = c.Deploy(context.Background(), "test user 2", dopts, &agent.Archive{DeploymentID: bw.MustGenerateID(), Peer: p})
		require.ErrorContains(t, err, "is already deploying")
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

	t.Run("a failed deploy must not block later deploys when the result observer is stalled", func(t *testing.T) {
		workdir := testingx.TempDir(t)
		p := agent.NewPeer("node1")
		// nothing ever reads from results, simulating a hung consumer on the result bus.
		results := make(chan *deployment.DeployResult)
		c := deployment.New(
			p,
			deployment.NewDirective(),
			deployment.CoordinatorOptionRoot(workdir),
			deployment.CoordinatorOptionStorage(storage.NoopRegistry{Err: context.DeadlineExceeded}),
			deployment.CoordinatorOptionDeployResults(results),
		)
		dopts := &agent.DeployOptions{Concurrency: 1, SilenceDeployLogs: true, Timeout: int64(time.Minute)}

		// mirrors bootstrap, which abandons the deploy context when Deploy returns an error.
		ctx, cancel := context.WithCancel(context.Background())
		_, err := c.Deploy(ctx, "test user", dopts, &agent.Archive{DeploymentID: bw.MustGenerateID(), Peer: p})
		require.Error(t, err)
		cancel()

		require.Eventually(t, func() bool {
			deploys, err := c.Deployments()
			require.NoError(t, err)
			return len(deploys) == 1 && deploys[0].Stage == agent.Deploy_Failed
		}, time.Second, 10*time.Millisecond)

		require.Eventually(t, func() bool {
			ctx, cancel := context.WithCancel(context.Background())
			defer cancel()
			_, err := c.Deploy(ctx, "test user", dopts, &agent.Archive{DeploymentID: bw.MustGenerateID(), Peer: p})
			return !strings.Contains(fmt.Sprint(err), "is already deploying")
		}, time.Second, 10*time.Millisecond)
	})
}

func TestMain(m *testing.M) {
	log.SetFlags(log.Flags() | log.Lshortfile)
	log.SetOutput(io.Discard)
	code := m.Run()
	testingx.Cleanup()
	os.Exit(code)
}
