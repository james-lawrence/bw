package bwfs_test

import (
	"io"
	"log"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"

	. "github.com/james-lawrence/bw/directives/bwfs"
	"github.com/james-lawrence/bw/internal/testingx"
)

// tempdir creates a directory relative to the working directory, the executer resolves paths relative to its root.
func tempdir(t *testing.T) string {
	t.Helper()
	tmpdir, err := os.MkdirTemp(".", "test")
	require.NoError(t, err)
	t.Cleanup(func() { os.RemoveAll(tmpdir) })
	return tmpdir
}

func newExecer() Executer {
	return New(log.New(io.Discard, "TEST ", log.LstdFlags), ".fixtures")
}

func TestShouldProperlyCopyADirectory(t *testing.T) {
	tmpdir := tempdir(t)
	archive := Archive{
		Owner: os.Getenv("USER"),
		Group: os.Getenv("USER"),
		Mode:  0766,
		Path:  filepath.Join(tmpdir, "sample-directory"),
		URI:   "sample-directory",
	}
	require.NoError(t, newExecer().Execute(archive))
	require.DirExists(t, archive.Path)
	testingx.RequireFilePermissions(t, archive.Path, os.FileMode(archive.Mode))
	require.DirExists(t, filepath.Join(archive.Path, "dir1"))
	require.DirExists(t, filepath.Join(archive.Path, "dir2"))
	require.FileExists(t, filepath.Join(archive.Path, "sample-file.txt"))
	require.FileExists(t, filepath.Join(archive.Path, "dir1", "sample-file.txt"))
	require.FileExists(t, filepath.Join(archive.Path, "dir2", "sample-file.txt"))
	testingx.RequireFilePermissions(t, archive.Path, os.FileMode(archive.Mode))
}

func TestShouldProperlyCopyAFile(t *testing.T) {
	for _, mode := range []uint32{0666, 0600} {
		t.Run(os.FileMode(mode).String(), func(t *testing.T) {
			tmpdir := tempdir(t)
			archive := Archive{
				Owner: os.Getenv("USER"),
				Group: os.Getenv("USER"),
				Mode:  mode,
				Path:  filepath.Join(tmpdir, "sample-file.txt"),
				URI:   "sample-file.txt",
			}
			require.NoError(t, newExecer().Execute(archive))
			require.FileExists(t, archive.Path)
			testingx.RequireFilePermissions(t, archive.Path, os.FileMode(archive.Mode))
		})
	}
}
