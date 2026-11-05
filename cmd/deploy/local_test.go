package deploy_test

import (
	"context"
	"net"
	"os"
	"path/filepath"
	"sync"
	"testing"

	"github.com/james-lawrence/bw"
	"github.com/james-lawrence/bw/agenttestutil"
	"github.com/james-lawrence/bw/cmd/deploy"
	"github.com/james-lawrence/bw/internal/rsax"
	"github.com/stretchr/testify/require"
)

func TestLocally(t *testing.T) {
	// a local deploy must run the directives from the .remote directory of the unpacked archive,
	// previously the archive was unpacked into a directory the deployment never looked at.
	// no agent is involved, the client is only configured to provide the environment.
	t.Run("runs the remote directives", func(t *testing.T) {
		l, err := net.Listen("tcp", "127.0.0.1:0")
		require.NoError(t, err)
		addr := l.Addr().(*net.TCPAddr)
		require.NoError(t, l.Close())

		seed := rsax.DeterministcSeed("bw.test", "locally", "client")
		client := sync.OnceValues(func() ([]byte, error) { return rsax.Deterministic(seed, rsax.UnsafeBits()) })
		require.NoError(t, agenttestutil.Client(t, client, addr))

		marker := filepath.Join(t.TempDir(), "marker")
		remote := filepath.Join(bw.DefaultDeployspaceDir, ".remote")
		require.NoError(t, os.MkdirAll(remote, 0700))
		require.NoError(t, os.WriteFile(filepath.Join(remote, "00_marker.bwcmd"), []byte("- command: touch "+marker+"\n"), 0600))

		require.NoError(t, deploy.Locally(&deploy.Context{
			Environment: agenttestutil.Environment,
			Context:     context.Background(),
		}, false))
		require.FileExists(t, marker)
	})

	t.Run("runs the local directives", func(t *testing.T) {
		l, err := net.Listen("tcp", "127.0.0.1:0")
		require.NoError(t, err)
		addr := l.Addr().(*net.TCPAddr)
		require.NoError(t, l.Close())

		seed := rsax.DeterministcSeed("bw.test", "locally", "client")
		client := sync.OnceValues(func() ([]byte, error) { return rsax.Deterministic(seed, rsax.UnsafeBits()) })
		require.NoError(t, agenttestutil.Client(t, client, addr))

		marker := filepath.Join(t.TempDir(), "marker")
		local := filepath.Join(bw.DefaultDeployspaceDir, ".local")
		require.NoError(t, os.MkdirAll(local, 0700))
		require.NoError(t, os.WriteFile(filepath.Join(local, "00_marker.bwcmd"), []byte("- command: touch "+marker+"\n"), 0600))

		require.NoError(t, deploy.Locally(&deploy.Context{
			Environment: agenttestutil.Environment,
			Context:     context.Background(),
		}, false))
		require.FileExists(t, marker)
	})

	t.Run("fails when a remote directive fails", func(t *testing.T) {
		l, err := net.Listen("tcp", "127.0.0.1:0")
		require.NoError(t, err)
		addr := l.Addr().(*net.TCPAddr)
		require.NoError(t, l.Close())

		seed := rsax.DeterministcSeed("bw.test", "locally", "client")
		client := sync.OnceValues(func() ([]byte, error) { return rsax.Deterministic(seed, rsax.UnsafeBits()) })
		require.NoError(t, agenttestutil.Client(t, client, addr))

		remote := filepath.Join(bw.DefaultDeployspaceDir, ".remote")
		require.NoError(t, os.MkdirAll(remote, 0700))
		require.NoError(t, os.WriteFile(filepath.Join(remote, "00_fail.bwcmd"), []byte("- command: \"false\"\n"), 0600))

		require.Error(t, deploy.Locally(&deploy.Context{
			Environment: agenttestutil.Environment,
			Context:     context.Background(),
		}, false))
	})
}
