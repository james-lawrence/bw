// Package blobfs provides writable file systems backed by local directories, google cloud storage
// and aws s3. Credentials come from the local credential chain of each provider.
package blobfs

import (
	"context"
	"fmt"
	"io"
	"io/fs"
	"net/http"
	"net/url"
	"os"
	"path"
	"strings"
	"time"

	"github.com/pkg/errors"
)

// FS is a file system which can additionally be written to. names follow the io/fs rules
// (slash separated, unrooted) and are relative to the location the FS was opened at.
type FS interface {
	fs.FS
	// OpenFile returns a writer for name, replacing any existing content. the content is committed
	// when the writer is closed, Close must be checked for errors. if ctx is cancelled before Close
	// the content is discarded.
	OpenFile(ctx context.Context, name string) (io.WriteCloser, error)
}

// Open the file system for the given location: gs://bucket/prefix, s3://bucket/prefix, or a local directory.
func Open(ctx context.Context, location string) (FS, error) {
	u, err := url.Parse(location)
	if err != nil || u.Scheme == "" || len(u.Scheme) == 1 {
		return NewDir(location), nil
	}

	switch u.Scheme {
	case "gs":
		return NewGCS(ctx, u.Host, u.Path)
	case "s3":
		return NewS3(u.Host, u.Path)
	case "file":
		return NewDir(u.Path), nil
	default:
		return nil, errors.Errorf("unsupported location %q, expected gs://, s3:// or a directory", location)
	}
}

func key(prefix, name string) (string, error) {
	if !fs.ValidPath(name) || name == "." {
		return "", &fs.PathError{Op: "key", Path: name, Err: fs.ErrInvalid}
	}

	return path.Join(strings.Trim(prefix, "/"), name), nil
}

// httpFile is the read side shared by the object stores, the object is fetched when opened.
type httpFile struct {
	name string
	resp *http.Response
}

func openHTTP(c *http.Client, name string, req *http.Request) (fs.File, error) {
	resp, err := c.Do(req)
	if err != nil {
		return nil, &fs.PathError{Op: "open", Path: name, Err: err}
	}

	switch {
	case resp.StatusCode == http.StatusNotFound:
		resp.Body.Close()
		return nil, &fs.PathError{Op: "open", Path: name, Err: fs.ErrNotExist}
	case resp.StatusCode/100 != 2:
		resp.Body.Close()
		return nil, &fs.PathError{Op: "open", Path: name, Err: errors.New(resp.Status)}
	}

	return httpFile{name: name, resp: resp}, nil
}

func (t httpFile) Read(b []byte) (int, error) { return t.resp.Body.Read(b) }
func (t httpFile) Close() error               { return t.resp.Body.Close() }
func (t httpFile) Stat() (fs.FileInfo, error) {
	return httpInfo{name: path.Base(t.name), size: t.resp.ContentLength}, nil
}

type httpInfo struct {
	name string
	size int64
}

func (t httpInfo) Name() string       { return t.name }
func (t httpInfo) Size() int64        { return t.size }
func (t httpInfo) Mode() fs.FileMode  { return 0444 }
func (t httpInfo) ModTime() time.Time { return time.Time{} }
func (t httpInfo) IsDir() bool        { return false }
func (t httpInfo) Sys() any           { return nil }

// commitWriter buffers writes to a temporary file, commit is called with the content when closed.
// the object stores require the content length up front, so the content cannot be streamed.
type commitWriter struct {
	ctx    context.Context
	f      *os.File
	commit func(r io.Reader, size int64) error
}

func newCommitWriter(ctx context.Context, commit func(r io.Reader, size int64) error) (*commitWriter, error) {
	f, err := os.CreateTemp("", "blobfs-*")
	if err != nil {
		return nil, err
	}

	return &commitWriter{ctx: ctx, f: f, commit: commit}, nil
}

func (t *commitWriter) Write(b []byte) (int, error) { return t.f.Write(b) }

func (t *commitWriter) Close() (err error) {
	defer os.Remove(t.f.Name())
	defer t.f.Close()

	if err = t.ctx.Err(); err != nil {
		return err
	}

	size, err := t.f.Seek(0, io.SeekCurrent)
	if err != nil {
		return err
	}

	if _, err = t.f.Seek(0, io.SeekStart); err != nil {
		return err
	}

	return t.commit(t.f, size)
}

func put(c *http.Client, req *http.Request, r io.Reader, size int64, desc string) error {
	req.Body = io.NopCloser(r)
	req.ContentLength = size

	resp, err := c.Do(req)
	if err != nil {
		return errors.Wrapf(err, "upload failed: %s", desc)
	}
	defer resp.Body.Close()

	if resp.StatusCode/100 != 2 {
		msg, _ := io.ReadAll(io.LimitReader(resp.Body, 1024))
		return fmt.Errorf("upload failed: %s: %s: %s", desc, resp.Status, msg)
	}

	return nil
}
