package certificatecache_test

import (
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/james-lawrence/bw/agent"
	"github.com/james-lawrence/bw/certificatecache"
	"github.com/stretchr/testify/require"
)

const (
	testServerName = "localhost"
	testToken      = "self-signed-test-token"
)

func writeagentconfig(t testing.TB) (path string, c agent.Config) {
	t.Helper()

	path = filepath.Join(t.TempDir(), "agent.yml")
	raw := "servername: " + testServerName + "\nclusterTokens:\n  - " + testToken + "\n"
	require.NoError(t, os.WriteFile(path, []byte(raw), 0600))

	c.ServerName = testServerName
	c.ClusterTokens = []string{testToken}

	return path, c
}

// bootstrap the self signed certificate the same way the agent daemon does.
func bootstrap(t testing.TB, c agent.Config, dir string) {
	t.Helper()

	ring, err := c.Keyring()
	require.NoError(t, err)
	require.NoError(t, certificatecache.AutomaticTLSAgent(ring.GetPrimaryKey(), c.ServerName, dir))
}

func TestFromConfigSelfSigned(t *testing.T) {
	t.Run("generates the same certificate as the agent bootstrap", func(t *testing.T) {
		cfg, c := writeagentconfig(t)
		expected, actual := t.TempDir(), t.TempDir()

		bootstrap(t, c, expected)
		require.NoError(t, certificatecache.FromConfig(actual, certificatecache.ModeSelfSigned, cfg, 0, nil))

		want, err := os.ReadFile(filepath.Join(expected, certificatecache.DefaultTLSSelfSignedCertServer))
		require.NoError(t, err)
		got, err := os.ReadFile(filepath.Join(actual, certificatecache.DefaultTLSSelfSignedCertServer))
		require.NoError(t, err)
		require.Equal(t, want, got, "seed and domain derived from the configuration must match the bootstrap")
	})

	t.Run("does not regenerate a valid certificate", func(t *testing.T) {
		cfg, c := writeagentconfig(t)
		dir := t.TempDir()
		path := filepath.Join(dir, certificatecache.DefaultTLSSelfSignedCertServer)

		bootstrap(t, c, dir)
		before, err := os.Stat(path)
		require.NoError(t, err)
		contents, err := os.ReadFile(path)
		require.NoError(t, err)

		require.NoError(t, certificatecache.FromConfig(dir, certificatecache.ModeSelfSigned, cfg, 0, nil))

		after, err := os.Stat(path)
		require.NoError(t, err)
		require.True(t, before.ModTime().Equal(after.ModTime()), "certificate file was rewritten")
		current, err := os.ReadFile(path)
		require.NoError(t, err)
		require.Equal(t, contents, current)
	})

	t.Run("never writes the signed certificate", func(t *testing.T) {
		cfg, _ := writeagentconfig(t)
		dir := t.TempDir()

		require.NoError(t, certificatecache.FromConfig(dir, certificatecache.ModeSelfSigned, cfg, 0, nil))

		require.NoFileExists(t, filepath.Join(dir, certificatecache.DefaultTLSCertServer))
		require.FileExists(t, filepath.Join(dir, certificatecache.DefaultTLSSelfSignedCertServer))
	})

	t.Run("disabled mode does not touch the credentials directory", func(t *testing.T) {
		cfg, _ := writeagentconfig(t)
		dir := t.TempDir()

		require.NoError(t, certificatecache.FromConfig(dir, certificatecache.ModeDisabled, cfg, time.Hour, nil))

		entries, err := os.ReadDir(dir)
		require.NoError(t, err)
		require.Empty(t, entries)
	})
}
