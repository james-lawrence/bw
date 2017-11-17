// Package uploads provides implementations for uploading archives.
// current implementations: fs (filesystem), aws s3.
package uploads

import (
	"crypto/sha256"
	"hash"
	"io"

	yaml "gopkg.in/yaml.v1"

	"github.com/pkg/errors"
)

const (
	fileProtocol = "local"
	s3Protocol   = "s3"
)

// Protocol builds io.WriteCloser given the expected size of the upload.
type Protocol interface {
	NewUpload(uid []byte, bytes uint64) (Uploader, error)
}

// ProtocolFunc - pure function protocol
type ProtocolFunc func(uid []byte, bytes uint64) (Uploader, error)

// NewUpload - implements Protocol
func (t ProtocolFunc) NewUpload(uid []byte, bytes uint64) (Uploader, error) {
	return t(uid, bytes)
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

// ProtocolFromConfig builds an upload protocol from a configuration.
func ProtocolFromConfig(protocol string, serialized []byte) (_ Protocol, err error) {
	switch protocol {
	case fileProtocol:
		var p Local
		return newProtocolFromConfig(serialized, &p)
	case s3Protocol:
		return newS3PFromConfig(serialized)
	default:
		return ProtocolFunc(
			func(uid []byte, _ uint64) (Uploader, error) {
				return NewTempFileUploader()
			},
		), nil
	}
}

func newProtocolFromConfig(serialized []byte, v Protocol) (_ Protocol, err error) {
	return v, errors.WithStack(yaml.Unmarshal(serialized, v))
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

func upload(src io.Reader, sha hash.Hash, dst io.Writer) (hash.Hash, error) {
	crc := sha256.New()
	_, err := io.Copy(io.MultiWriter(sha, crc, dst), src)
	return crc, errors.WithStack(err)
}
