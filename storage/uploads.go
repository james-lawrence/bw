package storage

import (
	"crypto/sha256"
	"hash"
	"io"
	"io/ioutil"

	yaml "gopkg.in/yaml.v2"

	"github.com/pkg/errors"
)

type uploadConfig interface {
	Uploader() (UploadProtocol, error)
}

// UploadProtocol builds io.WriteCloser given the expected size of the upload.
type UploadProtocol interface {
	NewUpload(bytes uint64) (Uploader, error)
}

// Uploader upload a file using the underlying protocol.
type Uploader interface {
	// Upload writes a chunk of data to the underlying storage
	// returning a checksum of the chunk or an error.
	Upload(io.Reader) (hash.Hash, error)

	// Info returns the result of the upload. this includes the overall checksum of the
	// file, the string uri of its location or an error.
	Info() (hash.Hash, string, error)
}

func uploadFromFile(path string, p uploadConfig) (_ UploadProtocol, err error) {
	var (
		b []byte
	)

	if b, err = ioutil.ReadFile(path); err != nil {
		return nil, errors.WithStack(err)
	}

	if err = errors.WithStack(yaml.Unmarshal(b, p)); err != nil {
		return nil, err
	}

	return p.Uploader()
}

func newProtocolFromConfig(serialized []byte, v UploadProtocol) (_ UploadProtocol, err error) {
	return v, errors.WithStack(yaml.Unmarshal(serialized, v))
}

// NewNoopUpload utility helper for returning a uploader that does nothing but
// return the specified error.
func NewNoopUpload(err error) Uploader {
	return errUploader{err: err}
}

type errUploader struct {
	err error
}

// Upload writes a chunk of data to the underlying storage
// returning a checksum of the chunk or an error.
func (t errUploader) Upload(io.Reader) (hash.Hash, error) {
	return nil, t.err
}

// Info returns the result of the upload. this includes the overall checksum of the
// file, the string uri of its location or an error.
func (t errUploader) Info() (hash.Hash, string, error) {
	return nil, "", t.err
}

// NewErrUploadProtocol upload protocol builder that returns an error
func NewErrUploadProtocol(err error) UploadProtocol {
	return errUploadProtocol{err: err}
}

type errUploadProtocol struct {
	err error
}

func (t errUploadProtocol) NewUpload(bytes uint64) (Uploader, error) {
	return nil, t.err
}

func upload(src io.Reader, sha hash.Hash, dst io.Writer) (hash.Hash, error) {
	crc := sha256.New()
	_, err := io.Copy(io.MultiWriter(sha, crc, dst), src)
	return crc, errors.WithStack(err)
}
