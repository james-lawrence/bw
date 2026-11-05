package agentutil_test

import (
	"fmt"
	"io"
	"log"
	"os"
	"testing"
	"time"

	. "github.com/james-lawrence/bw/agentutil"
	"github.com/stretchr/testify/require"
)

func TestMain(m *testing.M) {
	log.SetOutput(io.Discard)
	os.Exit(m.Run())
}

// dircleanupRoot creates the root directory for the directory cleanup tests, removed when the test completes.
func dircleanupRoot(t *testing.T) string {
	t.Helper()
	root, err := os.MkdirTemp(".", "dircleanup-test")
	require.NoError(t, err)
	t.Cleanup(func() { os.RemoveAll(root) })
	return root
}

func makedir(t *testing.T, root string) string {
	t.Helper()
	d, err := os.MkdirTemp(root, "")
	require.NoError(t, err)
	return d
}

func TestKeepNewestN(t *testing.T) {
	// should keep the n newest directories
	for i, c := range [][2]int{{5, 1}, {5, 2}, {5, 3}, {5, 4}, {5, 5}, {5, 6}} {
		m, n := c[0], c[1]
		t.Run(fmt.Sprintf("example %d", i+1), func(t *testing.T) {
			root := dircleanupRoot(t)
			if n > m {
				n = m
			}
			dirs := make([]string, 0, n)
			for i := 0; i < m; i++ {
				dirs = append(dirs, makedir(t, root))
				time.Sleep(30 * time.Millisecond)
			}

			MaybeClean(KeepNewestN(n))(Dirs(root))

			for _, d := range dirs[:len(dirs)-n] {
				require.NoDirExists(t, d)
			}
			for _, d := range dirs[len(dirs)-n:] {
				require.DirExists(t, d)
			}
		})
	}

	t.Run("should keep the n newest directories", func(t *testing.T) {
		root := dircleanupRoot(t)
		d1 := makedir(t, root)
		time.Sleep(30 * time.Millisecond)
		d2 := makedir(t, root)
		time.Sleep(30 * time.Millisecond)
		d3 := makedir(t, root)
		log.Println(d1, d2, d3)
		require.NoError(t, MaybeClean(KeepNewestN(1))(Dirs(root)))
		require.NoDirExists(t, d1)
		require.NoDirExists(t, d2)
		require.DirExists(t, d3)
	})
}

func TestKeepOldestN(t *testing.T) {
	// should keep the n oldest directories
	for i, c := range [][2]int{{5, 1}, {5, 2}, {5, 3}, {5, 4}, {5, 5}, {5, 6}} {
		m, n := c[0], c[1]
		t.Run(fmt.Sprintf("example %d", i+1), func(t *testing.T) {
			root := dircleanupRoot(t)
			if n > m {
				n = m
			}
			dirs := make([]string, 0, n)
			for i := 0; i < m; i++ {
				dirs = append(dirs, makedir(t, root))
				// necessary to ensure order of dirs by creation date.
				time.Sleep(30 * time.Millisecond)
			}

			require.NoError(t, MaybeClean(KeepOldestN(n))(Dirs(root)))

			for _, d := range dirs[:n] {
				require.DirExists(t, d)
			}
			for _, d := range dirs[n:] {
				require.NoDirExists(t, d)
			}
		})
	}

	t.Run("should keep the n oldest directories", func(t *testing.T) {
		root := dircleanupRoot(t)
		d1 := makedir(t, root)
		time.Sleep(30 * time.Millisecond)
		d2 := makedir(t, root)
		time.Sleep(30 * time.Millisecond)
		d3 := makedir(t, root)

		require.NoError(t, MaybeClean(KeepOldestN(1))(Dirs(root)))
		require.DirExists(t, d1)
		require.NoDirExists(t, d2)
		require.NoDirExists(t, d3)
	})
}
