package certificatecache

import (
	"context"
	"path/filepath"
	"sync/atomic"
	"testing"
	"time"

	"github.com/james-lawrence/bw/internal/tlsx"
	"github.com/stretchr/testify/require"
)

type refresherFunc func() error

func (t refresherFunc) Refresh() error { return t() }

func writeservercert(t testing.TB, dir string, valid time.Duration) {
	t.Helper()

	tmpl, err := tlsx.X509Template(valid)
	require.NoError(t, err)

	_, der, err := tlsx.SelfSignedRSAGen(2048, tmpl)
	require.NoError(t, err)
	require.NoError(t, tlsx.WriteCertificateFile(filepath.Join(dir, DefaultTLSCertServer), der))
}

func TestCertBackgroundRefresh(t *testing.T) {
	t.Run("refreshes once the initial due elapses", func(t *testing.T) {
		called := make(chan struct{}, 1)
		path := filepath.Join(t.TempDir(), "tls.cert")

		go certbackgroundrefresh(t.Context(), path, refresherFunc(func() error {
			select {
			case called <- struct{}{}:
			default:
			}
			return nil
		}), time.Millisecond, 0)

		select {
		case <-called:
		case <-time.After(5 * time.Second):
			t.Fatal("refresh was never invoked")
		}
	})

	t.Run("does not refresh before the initial due", func(t *testing.T) {
		var calls atomic.Int32
		path := filepath.Join(t.TempDir(), "tls.cert")

		go certbackgroundrefresh(t.Context(), path, refresherFunc(func() error {
			calls.Add(1)
			return nil
		}), time.Hour, 0)

		time.Sleep(100 * time.Millisecond)
		require.Zero(t, calls.Load())
	})

	t.Run("cancelled context prevents any refresh", func(t *testing.T) {
		var calls atomic.Int32
		ctx, cancel := context.WithCancel(context.Background())
		cancel()

		for range 100 {
			certbackgroundrefresh(ctx, filepath.Join(t.TempDir(), "tls.cert"), refresherFunc(func() error {
				calls.Add(1)
				return nil
			}), 0, 0)
		}

		require.Zero(t, calls.Load())
	})

	t.Run("returns when the context is cancelled", func(t *testing.T) {
		ctx, cancel := context.WithCancel(context.Background())
		done := make(chan struct{})

		go func() {
			defer close(done)
			certbackgroundrefresh(ctx, filepath.Join(t.TempDir(), "tls.cert"), refresherFunc(func() error { return nil }), time.Hour, 0)
		}()

		cancel()

		select {
		case <-done:
		case <-time.After(5 * time.Second):
			t.Fatal("loop did not exit after cancellation")
		}
	})
}

func TestRefreshAutomatic(t *testing.T) {
	t.Run("missing certificate refreshes once then waits for the computed due", func(t *testing.T) {
		var calls atomic.Int32

		require.NoError(t, RefreshAutomatic(t.TempDir(), 0, refresherFunc(func() error {
			calls.Add(1)
			return nil
		})))
		require.EqualValues(t, 1, calls.Load())

		// a zero due would cause the background loop to refresh immediately and repeatedly.
		time.Sleep(200 * time.Millisecond)
		require.EqualValues(t, 1, calls.Load())
	})

	t.Run("valid certificate is not refreshed", func(t *testing.T) {
		var calls atomic.Int32
		dir := t.TempDir()
		writeservercert(t, dir, 8*time.Hour)

		require.NoError(t, RefreshAutomatic(dir, 0, refresherFunc(func() error {
			calls.Add(1)
			return nil
		})))

		time.Sleep(200 * time.Millisecond)
		require.Zero(t, calls.Load())
	})

	t.Run("expired certificate is refreshed immediately and not repeatedly", func(t *testing.T) {
		var calls atomic.Int32
		dir := t.TempDir()
		writeservercert(t, dir, time.Hour)

		require.NoError(t, RefreshAutomatic(dir, 2*time.Hour, refresherFunc(func() error {
			calls.Add(1)
			return nil
		})))
		require.EqualValues(t, 1, calls.Load())

		time.Sleep(200 * time.Millisecond)
		require.EqualValues(t, 1, calls.Load())
	})

	t.Run("initial refresh error is returned and nothing is started", func(t *testing.T) {
		var calls atomic.Int32

		err := RefreshAutomatic(t.TempDir(), 0, refresherFunc(func() error {
			calls.Add(1)
			return context.DeadlineExceeded
		}))
		require.ErrorIs(t, err, context.DeadlineExceeded)

		time.Sleep(100 * time.Millisecond)
		require.EqualValues(t, 1, calls.Load())
	})
}
