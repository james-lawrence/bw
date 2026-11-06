package deploy

import (
	"crypto/sha256"
	"testing"

	"github.com/james-lawrence/bw/agent"
	"github.com/stretchr/testify/require"
)

func TestVerifyUpload(t *testing.T) {
	sum := sha256.Sum256([]byte("archive"))
	valid := func() *agent.Archive {
		return &agent.Archive{Location: "magnet:?xt=1", Checksum: sum[:], DeploymentID: sum[:], Commit: "abc"}
	}

	require.NoError(t, verifyUpload(valid(), sum[:], "abc"))

	for name, mutate := range map[string]func(*agent.Archive) *agent.Archive{
		"nil archive":           func(*agent.Archive) *agent.Archive { return nil },
		"empty location":        func(a *agent.Archive) *agent.Archive { a.Location = ""; return a },
		"checksum mismatch":     func(a *agent.Archive) *agent.Archive { a.Checksum = []byte("other"); return a },
		"deployment id differs": func(a *agent.Archive) *agent.Archive { a.DeploymentID = []byte("other"); return a },
		"commit mismatch":       func(a *agent.Archive) *agent.Archive { a.Commit = "def"; return a },
	} {
		t.Run(name, func(t *testing.T) {
			require.Error(t, verifyUpload(mutate(valid()), sum[:], "abc"))
		})
	}
}

func TestArchivingPeers(t *testing.T) {
	archiver := agent.NewPeer("archiver")
	archiver.Archive = true
	plain := agent.NewPeer("plain")

	got := archivingPeers(archiver, plain)
	require.Len(t, got, 1)
	require.Equal(t, "archiver", got[0].Name)
	require.Empty(t, archivingPeers(plain))
}
