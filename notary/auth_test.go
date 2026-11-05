package notary

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/james-lawrence/bw"
	"github.com/james-lawrence/bw/internal/rsax"
	"github.com/james-lawrence/bw/internal/testingx"
)

func TestNewAutoSignerPath(t *testing.T) {
	t.Run("should succeed when no key exists", func(t *testing.T) {
		_, err := newAutoSignerPath(
			filepath.Join(testingx.TempDir(t), bw.DefaultNotaryKey),
			"",
			rsax.UnsafeAuto,
		)
		require.NoError(t, err)
	})

	t.Run("should fail when unable to write to disk", func(t *testing.T) {
		tmp := testingx.TempDir(t)
		require.NoError(t, os.Chmod(tmp, 0400))
		_, err := newAutoSignerPath(
			filepath.Join(tmp, bw.DefaultNotaryKey),
			"",
			rsax.UnsafeAuto,
		)
		require.Error(t, err)
	})
}
