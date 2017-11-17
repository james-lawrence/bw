package storage

import (
	"crypto/sha256"
	"encoding/base64"
	"fmt"
	"hash"
	"io"
	"io/ioutil"
	"os"
	"path/filepath"

	"github.com/pkg/errors"
)

// Local storage protocol
type Local struct {
	Directory string
}

// NewUpload upload to a local directory.
func (t Local) NewUpload(uid []byte, bytes uint64) (Uploader, error) {
	var (
		err error
		dst *os.File
	)

	if dst, err = os.Create(filepath.Join(t.Directory, base64.URLEncoding.EncodeToString(uid))); err != nil {
		return nil, errors.WithStack(err)
	}

	return File{
		sha: sha256.New(),
		dst: dst,
	}, nil
}

// NewTempFileUploader - uploads archive to local storage. useful for testing.
func NewTempFileUploader() (File, error) {
	var (
		err error
		dst *os.File
	)

	if dst, err = ioutil.TempFile("", "prefix"); err != nil {
		return File{}, errors.Wrap(err, "failed to create temporary file")
	}

	return File{
		sha: sha256.New(),
		dst: dst,
	}, nil
}

// File ...
type File struct {
	sha hash.Hash
	dst *os.File
}

// Upload ...
func (t File) Upload(r io.Reader) (hash.Hash, error) {
	return upload(r, t.sha, t.dst)
}

// Info ...
func (t File) Info() (hash.Hash, string, error) {
	if err := t.dst.Sync(); err != nil {
		return nil, "", errors.Wrap(err, "failed to sync to disk")
	}

	if err := t.dst.Close(); err != nil {
		return nil, "", errors.Wrap(err, "failed to close upload")
	}

	return t.sha, fmt.Sprintf("file://%s", t.dst.Name()), nil
}
