package iox

import (
	"io"

	"github.com/pkg/errors"
)

// IgnoreEOF returns nil if err is io.EOF
func IgnoreEOF(err error) error {
	if errors.Cause(err) != io.EOF {
		return err
	}

	return nil
}

// Error return just the error from an IO call ignoring the number of bytes.
func Error(_ int64, err error) error {
	return err
}

type errReader struct {
	error
}

func (t errReader) Read([]byte) (int, error) {
	return 0, t
}

// ErrReader returns an io.Reader that returns the provided error.
func ErrReader(err error) io.Reader {
	return errReader{err}
}

// Rewind an io.Seeker
func Rewind(o io.Seeker) error {
	_, err := o.Seek(0, io.SeekStart)
	return err
}
