package blobfs

import (
	"context"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"sync"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestDir(t *testing.T) {
	d := NewDir(t.TempDir())
	write(t, d, "a/b.txt", "hello")
	f, err := d.Open("a/b.txt")
	require.NoError(t, err)
	raw, _ := io.ReadAll(f)
	require.Equal(t, "hello", string(raw))
	_, err = d.OpenFile(context.Background(), "../escape")
	require.Error(t, err)

	ctx, cancel := context.WithCancel(context.Background())
	w, err := d.OpenFile(ctx, "cancelled.txt")
	require.NoError(t, err)
	_, _ = w.Write([]byte("x"))
	cancel()
	require.Error(t, w.Close())
	_, err = d.Open("cancelled.txt")
	require.Error(t, err)
}

func write(t *testing.T, f FS, name, content string) {
	w, err := f.OpenFile(context.Background(), name)
	require.NoError(t, err)
	_, err = io.WriteString(w, content)
	require.NoError(t, err)
	require.NoError(t, w.Close())
}

func TestGCS(t *testing.T) {
	var (
		mu       sync.Mutex
		received = map[string]string{}
	)

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		mu.Lock()
		defer mu.Unlock()
		if r.Method == http.MethodPost {
			body, _ := io.ReadAll(r.Body)
			received[r.URL.Query().Get("name")] = string(body)
			return
		}
		io.WriteString(w, "downloaded")
	}))
	defer srv.Close()

	g := &GCS{client: srv.Client(), endpoint: srv.URL, bucket: "bucket", prefix: "a/b/", ctx: context.Background()}
	write(t, g, "current.meta", "meta")
	require.Equal(t, map[string]string{"a/b/current.meta": "meta"}, received)

	f, err := g.Open("current.meta")
	require.NoError(t, err)
	raw, _ := io.ReadAll(f)
	require.Equal(t, "downloaded", string(raw))
}

func TestOpenRejectsUnknownSchemes(t *testing.T) {
	_, err := Open(context.Background(), "ftp://host/dir")
	require.Error(t, err)
}

func TestOpenDirectory(t *testing.T) {
	dir := t.TempDir()
	for _, location := range []string{dir, "file://" + dir} {
		f, err := Open(context.Background(), location)
		require.NoError(t, err)
		write(t, f, "current.meta", "meta")
		raw, err := os.ReadFile(filepath.Join(dir, "current.meta"))
		require.NoError(t, err)
		require.Equal(t, "meta", string(raw))
	}
}
