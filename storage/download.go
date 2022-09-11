package storage

import (
	"archive/tar"
	"bytes"
	"compress/gzip"
	"context"
	"io"
	"strings"
	"time"

	"github.com/james-lawrence/bw/agent"
	"github.com/james-lawrence/bw/internal/errorsx"
	"github.com/pkg/errors"
)

// DownloadFactory ...
type DownloadFactory interface {
	New(string) Downloader
}

// Downloader ...
type Downloader interface {
	Download(context.Context, *agent.Archive) io.ReadCloser
}

func newDownload(rc io.ReadCloser) rcDownload {
	return rcDownload{rc}
}

type rcDownload struct {
	io.ReadCloser
}

func (t rcDownload) Download(context.Context, *agent.Archive) io.ReadCloser {
	return t.ReadCloser
}

// DownloadProtocol ...
type DownloadProtocol interface {
	Protocol() string
	New() Downloader
}

// Option for a download registry.
type Option func(*Registry)

// OptionProtocols set the protocols available for this registry.
func OptionProtocols(protocols ...DownloadProtocol) Option {
	pp := make(map[string]DownloadProtocol, len(protocols))
	for _, p := range protocols {
		pp[p.Protocol()] = p
	}

	return func(r *Registry) {
		r.protocols = pp
	}
}

// New create a new downloader registry.
func New(options ...Option) (r Registry) {
	for _, opt := range options {
		opt(&r)
	}

	return r
}

// Registry of protocols for downloads.
type Registry struct {
	protocols map[string]DownloadProtocol
}

// New connect to the specified location by creating a io.ReadCloser
func (t Registry) New(location string) Downloader {
	for prefix, p := range t.protocols {
		if strings.HasPrefix(location, prefix) {
			return p.New()
		}
	}

	return newDownload(newErrReader(errors.Errorf("unknown protocol: [%s]", location)))
}

// NoopRegistry simple registry that returns an err if set otherwise
// an empty archive.
type NoopRegistry struct {
	Err error
}

// New ...
func (t NoopRegistry) New(location string) Downloader {
	if t.Err != nil {
		return newDownload(newErrReader(t.Err))
	}

	return newDownload(buildArchive([]byte{}))
}

func buildArchive(input []byte) io.ReadCloser {
	b := bytes.NewBuffer([]byte{})
	gzw := gzip.NewWriter(b)
	tw := tar.NewWriter(gzw)
	header := tar.Header{
		Name:     "example",
		ModTime:  time.Now(),
		Typeflag: tar.TypeReg,
		Size:     int64(len(input)),
		Mode:     0700,
		Uid:      0,
		Gid:      0,
		Uname:    "root",
		Gname:    "root",
	}

	if err := tw.WriteHeader(&header); err != nil {
		return newDownload(newErrReader(err))
	}

	_, err := tw.Write(input)
	if err = errorsx.Compact(err, tw.Flush(), tw.Close(), gzw.Flush(), gzw.Close()); err != nil {
		return newErrReader(errors.New("failed to build archive"))
	}

	return io.NopCloser(b)
}
