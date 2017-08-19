package agent

import (
	"crypto/sha256"
	"fmt"
	"hash"
	"io"
	"io/ioutil"
	"os"

	"github.com/pkg/errors"
)

// NewFileUploader - uploads archive to local storage.
func NewFileUploader() (UploadFile, error) {
	var (
		err error
		dst *os.File
	)

	if dst, err = ioutil.TempFile("", "prefix"); err != nil {
		return UploadFile{}, errors.Wrap(err, "failed to create temporary file")
	}

	return UploadFile{
		sha: sha256.New(),
		dst: dst,
	}, nil
}

// UploadFile ...
type UploadFile struct {
	sha hash.Hash
	dst *os.File
}

// Upload ...
func (t UploadFile) Upload(r io.Reader) (hash.Hash, error) {
	crc := sha256.New()
	dst := io.MultiWriter(t.sha, t.dst)
	_, err := io.Copy(io.MultiWriter(dst, crc), r)
	return crc, err
}

// Info ...
func (t UploadFile) Info() (hash.Hash, string, error) {
	if err := t.dst.Sync(); err != nil {
		return nil, "", errors.Wrap(err, "failed to sync to disk")
	}

	if err := t.dst.Close(); err != nil {
		return nil, "", errors.Wrap(err, "failed to close upload")
	}

	return t.sha, fmt.Sprintf("file://%s", t.dst.Name()), nil
}
