package certificatecache_test

import (
	"errors"
	"path/filepath"
	"testing"
	"time"

	"github.com/james-lawrence/bw/certificatecache"
	"github.com/james-lawrence/bw/internal/tlsx"
	"github.com/stretchr/testify/require"
)

type refresherFunc func() error

func (t refresherFunc) Refresh() error { return t() }

func writecert(t testing.TB, path string, valid time.Duration) {
	t.Helper()

	tmpl, err := tlsx.X509Template(valid)
	require.NoError(t, err)

	key, der, err := tlsx.SelfSignedRSAGen(2048, tmpl)
	require.NoError(t, err)
	require.NoError(t, tlsx.WriteCertificateFile(path, der))
	_ = key
}

func TestRefreshExpired(t *testing.T) {
	t.Run("missing certificate forces a refresh", func(t *testing.T) {
		var calls int
		path := filepath.Join(t.TempDir(), "nested", "tls.cert")

		due, err := certificatecache.RefreshExpired(path, time.Now(), refresherFunc(func() error {
			calls++
			return nil
		}))
		require.NoError(t, err)
		require.Equal(t, 1, calls)
		require.Equal(t, time.Minute, due)
	})

	t.Run("missing certificate propagates the refresh error", func(t *testing.T) {
		expected := errors.New("boom")
		path := filepath.Join(t.TempDir(), "tls.cert")

		_, err := certificatecache.RefreshExpired(path, time.Now(), refresherFunc(func() error { return expected }))
		require.ErrorIs(t, err, expected)
	})

	t.Run("valid certificate is not refreshed", func(t *testing.T) {
		var calls int
		path := filepath.Join(t.TempDir(), "tls.cert")
		writecert(t, path, 8*time.Hour)

		due, err := certificatecache.RefreshExpired(path, time.Now(), refresherFunc(func() error {
			calls++
			return nil
		}))
		require.NoError(t, err)
		require.Zero(t, calls)
		require.InDelta(t, float64(2*time.Hour), float64(due), float64(time.Minute)) // expiration / 4
	})

	t.Run("due is capped at a day", func(t *testing.T) {
		path := filepath.Join(t.TempDir(), "tls.cert")
		writecert(t, path, 30*24*time.Hour)

		due, err := certificatecache.RefreshExpired(path, time.Now(), refresherFunc(func() error { return nil }))
		require.NoError(t, err)
		require.Equal(t, 24*time.Hour, due)
	})

	t.Run("due has a one minute floor", func(t *testing.T) {
		path := filepath.Join(t.TempDir(), "tls.cert")
		writecert(t, path, time.Hour)

		due, err := certificatecache.RefreshExpired(path, time.Now().Add(2*time.Hour), refresherFunc(func() error { return nil }))
		require.NoError(t, err)
		require.Equal(t, time.Minute, due)
	})

	t.Run("expired certificate is refreshed", func(t *testing.T) {
		var calls int
		path := filepath.Join(t.TempDir(), "tls.cert")
		writecert(t, path, time.Hour)

		_, err := certificatecache.RefreshExpired(path, time.Now().Add(2*time.Hour), refresherFunc(func() error {
			calls++
			return nil
		}))
		require.NoError(t, err)
		require.Equal(t, 1, calls)
	})

	t.Run("future offset triggers refresh before expiry", func(t *testing.T) {
		var calls int
		path := filepath.Join(t.TempDir(), "tls.cert")
		writecert(t, path, time.Hour)

		_, err := certificatecache.RefreshExpired(path, time.Now().Add(2*time.Hour), refresherFunc(func() error {
			calls++
			return nil
		}))
		require.NoError(t, err)
		require.Equal(t, 1, calls)
	})

	t.Run("invalid certificate returns an error", func(t *testing.T) {
		path := filepath.Join(t.TempDir(), "tls.cert")
		require.NoError(t, tlsx.WriteCertificateFile(path, []byte("garbage")))

		_, err := certificatecache.RefreshExpired(path, time.Now(), refresherFunc(func() error { return nil }))
		require.Error(t, err)
	})
}
