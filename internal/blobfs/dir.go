package blobfs

import (
	"context"
	"io"
	"io/fs"
	"os"
	"path/filepath"
)

// NewDir writes to a local directory.
func NewDir(root string) Dir {
	return Dir{root: root}
}

// Dir is a FS backed by a local directory.
type Dir struct {
	root string
}

func (t Dir) Open(name string) (fs.File, error) {
	return os.DirFS(t.root).Open(name)
}

func (t Dir) OpenFile(ctx context.Context, name string) (io.WriteCloser, error) {
	if !fs.ValidPath(name) || name == "." {
		return nil, &fs.PathError{Op: "write", Path: name, Err: fs.ErrInvalid}
	}

	dst := filepath.Join(t.root, filepath.FromSlash(name))
	if err := os.MkdirAll(filepath.Dir(dst), 0755); err != nil {
		return nil, err
	}

	f, err := os.CreateTemp(filepath.Dir(dst), ".write-*")
	if err != nil {
		return nil, err
	}

	return dirWriter{ctx: ctx, f: f, dst: dst}, nil
}

// dirWriter writes to a temporary file which replaces dst when closed.
type dirWriter struct {
	ctx context.Context
	f   *os.File
	dst string
}

func (t dirWriter) Write(b []byte) (int, error) { return t.f.Write(b) }

func (t dirWriter) Close() error {
	if err := t.ctx.Err(); err != nil {
		t.f.Close()
		os.Remove(t.f.Name())
		return err
	}

	if err := t.f.Close(); err != nil {
		os.Remove(t.f.Name())
		return err
	}

	return os.Rename(t.f.Name(), t.dst)
}
