package deploy_test

import (
	"bytes"
	"context"
	"net"
	"os"
	"path/filepath"
	"sync"
	"testing"

	"github.com/james-lawrence/bw"
	"github.com/james-lawrence/bw/agenttestutil"
	"github.com/james-lawrence/bw/archive"
	"github.com/james-lawrence/bw/cmd/deploy"
	"github.com/james-lawrence/bw/internal/rsax"
	"github.com/stretchr/testify/require"
)

type snapshotBuffer struct{ bytes.Buffer }

func (*snapshotBuffer) Close() error { return nil }

func TestSnapshot(t *testing.T) {
	// the snapshot is the archive that would be deployed, after the local directives have run.
	// no agent is involved, the client is only configured to provide the environment.
	t.Run("packs the deployspace after running the local directives", func(t *testing.T) {
		l, err := net.Listen("tcp", "127.0.0.1:0")
		require.NoError(t, err)
		addr := l.Addr().(*net.TCPAddr)
		require.NoError(t, l.Close())

		seed := rsax.DeterministcSeed("bw.test", "snapshot", "client")
		client := sync.OnceValues(func() ([]byte, error) { return rsax.Deterministic(seed, rsax.UnsafeBits()) })
		require.NoError(t, agenttestutil.Client(t, client, addr))

		remote := filepath.Join(bw.DefaultDeployspaceDir, ".remote")
		require.NoError(t, os.MkdirAll(remote, 0700))
		require.NoError(t, os.WriteFile(filepath.Join(remote, "00_noop.bwcmd"), []byte("- command: \"true\"\n"), 0600))

		var out snapshotBuffer
		require.NoError(t, deploy.Snapshot(deploy.Context{
			Environment: agenttestutil.Environment,
			Context:     context.Background(),
		}, &out))

		dst := t.TempDir()
		require.NoError(t, archive.Unpack(dst, &out))
		require.DirExists(t, filepath.Join(dst, ".remote"))
		require.FileExists(t, filepath.Join(dst, bw.EnvFile))
	})
}
