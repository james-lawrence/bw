package deploy_test

import (
	"context"
	"net"
	"os"
	"path/filepath"
	"sync"
	"testing"
	"time"

	"github.com/james-lawrence/bw"
	"github.com/james-lawrence/bw/agenttestutil"
	"github.com/james-lawrence/bw/cmd/deploy"
	"github.com/james-lawrence/bw/deployment"
	"github.com/james-lawrence/bw/internal/rsax"
	"github.com/stretchr/testify/require"
)

func TestInto(t *testing.T) {
	if testing.Short() {
		t.Skip("boots an agent, skipping in short mode")
	}

	// deploying must return once the agent completes the deploy.
	t.Run("runs the remote directives on the agent", func(t *testing.T) {
		l, err := net.Listen("tcp", "127.0.0.1:0")
		require.NoError(t, err)
		addr := l.Addr().(*net.TCPAddr)
		require.NoError(t, l.Close())

		seed := rsax.DeterministcSeed("bw.test", "into", "client")
		client := sync.OnceValues(func() ([]byte, error) { return rsax.Deterministic(seed, rsax.UnsafeBits()) })

		require.NoError(t, agenttestutil.Agent(t, client, addr))
		require.NoError(t, agenttestutil.Client(t, client, addr))
		require.NoError(t, agenttestutil.WaitQuorum(context.Background()))

		marker := filepath.Join(t.TempDir(), "marker")
		remote := filepath.Join(bw.DefaultDeployspaceDir, ".remote")
		require.NoError(t, os.MkdirAll(remote, 0700))
		require.NoError(t, os.WriteFile(filepath.Join(remote, "00_marker.bwcmd"), []byte("- command: touch "+marker+"\n"), 0600))

		ctx, cancel := context.WithCancel(context.Background())
		defer cancel()

		result := make(chan error, 1)
		go func() {
			result <- deploy.Into(&deploy.Context{
				Environment: agenttestutil.Environment,
				Insecure:    true,
				Attempts:    1,
				Heartbeat:   10 * time.Second,
				Filter:      deployment.AlwaysMatch,
				Context:     ctx,
				CancelFunc:  cancel,
				WaitGroup:   &sync.WaitGroup{},
			})
		}()

		select {
		case err := <-result:
			require.NoError(t, err)
		case <-time.After(20 * time.Second):
			t.Fatal("deploy did not return after the agent completed the deploy")
		}
		require.FileExists(t, marker)
	})

	// a failing remote directive must fail the deploy, and not hang waiting on the agent.
	t.Run("fails when a remote directive fails", func(t *testing.T) {
		l, err := net.Listen("tcp", "127.0.0.1:0")
		require.NoError(t, err)
		addr := l.Addr().(*net.TCPAddr)
		require.NoError(t, l.Close())

		seed := rsax.DeterministcSeed("bw.test", "into", "client")
		client := sync.OnceValues(func() ([]byte, error) { return rsax.Deterministic(seed, rsax.UnsafeBits()) })

		require.NoError(t, agenttestutil.Agent(t, client, addr))
		require.NoError(t, agenttestutil.Client(t, client, addr))
		require.NoError(t, agenttestutil.WaitQuorum(context.Background()))

		remote := filepath.Join(bw.DefaultDeployspaceDir, ".remote")
		require.NoError(t, os.MkdirAll(remote, 0700))
		require.NoError(t, os.WriteFile(filepath.Join(remote, "00_fail.bwcmd"), []byte("- command: \"false\"\n"), 0600))

		ctx, cancel := context.WithCancel(context.Background())
		defer cancel()

		result := make(chan error, 1)
		go func() {
			result <- deploy.Into(&deploy.Context{
				Environment: agenttestutil.Environment,
				Insecure:    true,
				Attempts:    1,
				Heartbeat:   10 * time.Second,
				Filter:      deployment.AlwaysMatch,
				Context:     ctx,
				CancelFunc:  cancel,
				WaitGroup:   &sync.WaitGroup{},
			})
		}()

		select {
		case err := <-result:
			require.Error(t, err)
		case <-time.After(20 * time.Second):
			t.Fatal("deploy did not return after the agent failed the deploy")
		}
	})
}

func TestForce(t *testing.T) {
	if testing.Short() {
		t.Skip("boots an agent, skipping in short mode")
	}

	// forcing an archive is pointless when no node archives deployments. the command must refuse before
	// uploading or announcing anything, and must not restart the cluster.
	t.Run("refuses when no node archives deployments", func(t *testing.T) {
		l, err := net.Listen("tcp", "127.0.0.1:0")
		require.NoError(t, err)
		addr := l.Addr().(*net.TCPAddr)
		require.NoError(t, l.Close())

		seed := rsax.DeterministcSeed("bw.test", "force", "client")
		client := sync.OnceValues(func() ([]byte, error) { return rsax.Deterministic(seed, rsax.UnsafeBits()) })

		require.NoError(t, agenttestutil.Agent(t, client, addr))
		require.NoError(t, agenttestutil.Client(t, client, addr))
		require.NoError(t, agenttestutil.WaitQuorum(context.Background()))

		require.NoError(t, os.MkdirAll(filepath.Join(bw.DefaultDeployspaceDir, ".remote"), 0700))

		ctx, cancel := context.WithCancel(context.Background())
		defer cancel()

		result := make(chan error, 1)
		go func() {
			result <- deploy.Force(&deploy.Context{
				Environment: agenttestutil.Environment,
				Insecure:    true,
				Heartbeat:   10 * time.Second,
				Filter:      deployment.AlwaysMatch,
				Context:     ctx,
				CancelFunc:  cancel,
				WaitGroup:   &sync.WaitGroup{},
			})
		}()

		select {
		case err := <-result:
			require.ErrorContains(t, err, "no nodes in the cluster archive deployments")
		case <-time.After(20 * time.Second):
			t.Fatal("force did not return")
		}

		// the agent must still be up.
		require.NoError(t, agenttestutil.WaitQuorum(context.Background()))
	})
}
