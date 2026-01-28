package notary

import (
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/james-lawrence/bw/internal/errorsx"
	"github.com/james-lawrence/bw/internal/rsax"
	"github.com/james-lawrence/bw/internal/sshx"
	"github.com/stretchr/testify/require"
)

func TestNewFile(t *testing.T) {
	t.Run("creates a new file", func(t *testing.T) {
		tempDir := t.TempDir()
		path := filepath.Join(tempDir, "test.key")

		f, err := newFile(path)
		require.NoError(t, err)
		require.NotNil(t, f)
		require.NoError(t, os.RemoveAll(path))
	})

	t.Run("creates a new file with parent directories", func(t *testing.T) {
		tempDir := t.TempDir()
		path := filepath.Join(tempDir, "keys", "test.key")

		f, err := newFile(path)
		require.NoError(t, err)
		require.NotNil(t, f)
		require.NoError(t, os.RemoveAll(path))
	})

	t.Run("update storage when the file is updated", func(t *testing.T) {
		var (
			tempDir   = t.TempDir()
			path      = filepath.Join(tempDir, "keys", "test.key")
			updpath   = filepath.Join(tempDir, "derp.keys")
			generated = errorsx.Must(rsax.UnsafeAuto())
			marshaled = errorsx.Must(sshx.PublicKey(generated))
			id        = sshx.FingerprintSHA256(marshaled)
			f         *file
			g         *Grant
			err       error
		)

		require.NoError(t, os.WriteFile(updpath, marshaled, 0600))

		f, err = newFile(path)
		require.NoError(t, err)
		require.NotNil(t, f)

		g, err = f.Lookup(id)
		require.ErrorIs(t, err, ErrFingerprintNotFound)
		require.Nil(t, g)

		require.NoError(t, CloneAuthorizationFile(updpath, path))

		require.Eventually(t, func() bool {
			g, err = f.Lookup(id)
			return err == nil && g != nil
		}, time.Second, 10*time.Millisecond)
	})

	t.Run("file watcher with rename-based clone", func(t *testing.T) {
		var (
			tempDir    = t.TempDir()
			path       = filepath.Join(tempDir, "keys", "test.key")
			updpath1   = filepath.Join(tempDir, "derp1.keys")
			updpath2   = filepath.Join(tempDir, "derp2.keys")
			generated1 = errorsx.Must(rsax.UnsafeAuto())
			generated2 = errorsx.Must(rsax.UnsafeAuto())
			marshaled1 = errorsx.Must(sshx.PublicKey(generated1))
			marshaled2 = errorsx.Must(sshx.PublicKey(generated2))
			id1        = sshx.FingerprintSHA256(marshaled1)
			id2        = sshx.FingerprintSHA256(marshaled2)
			f          *file
			g          *Grant
			err        error
		)

		require.NoError(t, os.WriteFile(updpath1, marshaled1, 0600))
		require.NoError(t, os.WriteFile(updpath2, marshaled2, 0600))

		f, err = newFile(path)
		require.NoError(t, err)
		require.NotNil(t, f)

		g, err = f.Lookup(id1)
		require.ErrorIs(t, err, ErrFingerprintNotFound)
		require.Nil(t, g)

		require.NoError(t, CloneAuthorizationFile(updpath1, path))

		require.Eventually(t, func() bool {
			g, err = f.Lookup(id1)
			return err == nil && g != nil
		}, time.Second, 10*time.Millisecond)

		require.NoError(t, CloneAuthorizationFile(updpath2, path))

		require.Eventually(t, func() bool {
			g, err = f.Lookup(id2)
			return err == nil && g != nil
		}, time.Second, 10*time.Millisecond)
	})
}
