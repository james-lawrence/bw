package peering_test

import (
	"context"
	"io"
	"log"
	"os"
	"path/filepath"
	"testing"

	"github.com/james-lawrence/bw/clustering"
	"github.com/james-lawrence/bw/clustering/clusteringtestutil"
	"github.com/stretchr/testify/require"

	. "github.com/james-lawrence/bw/clustering/peering"
)

func TestMain(m *testing.M) {
	log.SetOutput(io.Discard)
	os.Exit(m.Run())
}

func TestFileShouldPersistThePeersToAFile(t *testing.T) {
	tmpdir := t.TempDir()

	c := clusteringtestutil.NewMock(3)
	fs := File{Path: filepath.Join(tmpdir, "peers.yml")}
	require.NoError(t, fs.Snapshot(clustering.Peers(c)))

	peers, err := fs.Peers(context.Background())
	require.NoError(t, err)
	require.ElementsMatch(t, clustering.Peers(c), peers)
}
