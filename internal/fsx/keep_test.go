package fsx_test

import (
	"context"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/james-lawrence/bw/internal/fsx"
	"github.com/james-lawrence/bw/internal/iterx"
	"github.com/stretchr/testify/require"
)

func touch(t *testing.T, path string, age time.Duration) {
	require.NoError(t, os.WriteFile(path, nil, 0600))
	at := time.Now().Add(-age)
	require.NoError(t, os.Chtimes(path, at, at))
}

func TestKeepNewestN(t *testing.T) {
	root := t.TempDir()
	a, b, c := filepath.Join(root, "a"), filepath.Join(root, "b"), filepath.Join(root, "c")
	touch(t, a, 3*time.Hour)
	touch(t, b, 2*time.Hour)
	touch(t, c, 1*time.Hour)

	collect := func(n int) (r []string) {
		s := fsx.KeepNewestN(n, iterx.Slice(a, b, c))
		for p := range s.Each(context.Background()) {
			r = append(r, p)
		}
		require.NoError(t, s.Err())
		return r
	}

	require.ElementsMatch(t, []string{a, b}, collect(1))
	require.ElementsMatch(t, []string{a, b, c}, collect(0))
	require.Empty(t, collect(5))
}
