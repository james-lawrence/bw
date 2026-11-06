package bootstrap

import (
	"bytes"
	"context"
	"crypto/sha256"
	"io"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/james-lawrence/bw"
	"github.com/james-lawrence/bw/agent"
	"github.com/james-lawrence/bw/internal/testingx"
	"github.com/james-lawrence/bw/storage"
)

type testDownloadProtocol struct {
	data []byte
}

func (t testDownloadProtocol) Protocol() string { return "test" }
func (t testDownloadProtocol) New() storage.Downloader {
	return testDownloader(t)
}

type testDownloader struct{ data []byte }

func (t testDownloader) Download(context.Context, *agent.Archive) io.ReadCloser {
	return io.NopCloser(bytes.NewReader(t.data))
}

func forcedCommand(data []byte) *agent.DeployCommand {
	sum := sha256.Sum256(data)
	id := bw.MustGenerateID()
	archive := &agent.Archive{
		Location:     "test://archive",
		Checksum:     sum[:],
		DeploymentID: id[:],
	}
	return agent.DeployCommandDone("tester", archive.DeployOption, (&agent.DeployOptions{}).DeployOption)
}

func newTestFilesystem(t *testing.T, dl storage.DownloadProtocol) (Filesystem, string) {
	t.Helper()
	dir := testingx.TempDir(t)
	return NewFilesystem(agent.NewConfig(
		agent.ConfigOptionBootstrapArchive(dir, false),
		func(c *agent.Config) { c.Root = testingx.TempDir(t) },
	), nil, nil, dl), dir
}

func TestFilesystemCloneDownloadsMissingArchive(t *testing.T) {
	data := []byte("forced archive contents")

	t.Run("downloads and stores the archive when there is no local copy", func(t *testing.T) {
		fs, dir := newTestFilesystem(t, testDownloadProtocol{data: data})
		dc := forcedCommand(data)

		// stale marker from a previous archive must be replaced.
		require.NoError(t, os.WriteFile(filepath.Join(dir, "current.uploaded"), []byte("stale"), 0600))

		require.NoError(t, fs.clone(dc))

		stored, err := os.ReadFile(filepath.Join(dir, "current.tar.gz"))
		require.NoError(t, err)
		require.Equal(t, data, stored)

		meta, err := agent.ReadMetadata(filepath.Join(dir, "current.meta"))
		require.NoError(t, err)
		require.Equal(t, dc.Archive.DeploymentID, meta.Archive.DeploymentID)

		uploaded, err := agent.ReadMetadata(filepath.Join(dir, "current.uploaded"))
		require.NoError(t, err)
		require.Equal(t, dc.Archive.DeploymentID, uploaded.Archive.DeploymentID)
	})

	t.Run("checksum mismatch keeps the previous archive", func(t *testing.T) {
		fs, dir := newTestFilesystem(t, testDownloadProtocol{data: []byte("corrupted")})
		require.NoError(t, os.WriteFile(filepath.Join(dir, "current.tar.gz"), []byte("previous"), 0600))

		require.Error(t, fs.clone(forcedCommand(data)))

		stored, err := os.ReadFile(filepath.Join(dir, "current.tar.gz"))
		require.NoError(t, err)
		require.Equal(t, []byte("previous"), stored)
	})

	t.Run("without a download protocol a missing local copy is an error", func(t *testing.T) {
		fs, _ := newTestFilesystem(t, nil)
		require.Error(t, fs.clone(forcedCommand(data)))
	})
}
