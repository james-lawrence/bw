package notary

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/james-lawrence/bw/internal/errorsx"
	"github.com/james-lawrence/bw/internal/rsax"
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
		tempDir := t.TempDir()
		path := filepath.Join(tempDir, "keys", "test.key")

		updpath := filepath.Join(tempDir, "derp.keys")
		generated := errorsx.Must(rsax.UnsafeAuto())
		id := rsax.FingerprintSHA256(generated)
		require.NoError(t, os.WriteFile(updpath, generated, 0600))

		f, err := newFile(path)
		require.NoError(t, err)
		require.NotNil(t, f)

		g, err := f.Lookup(id)
		require.ErrorIs(t, err, ErrFingerprintNotFound)
		require.Nil(t, g)

		require.NoError(t, CloneAuthorizationFile(updpath, path))
		g, err = f.Lookup(id)
		require.NoError(t, err)
		require.Nil(t, g)
	})
}
