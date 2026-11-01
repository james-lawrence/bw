package iox

import (
	"io"
	"os"
	"sync/atomic"

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

type countingReader struct {
	io.Reader
	n *atomic.Uint64
}

func (t countingReader) Read(p []byte) (int, error) {
	n, err := t.Reader.Read(p)
	t.n.Add(uint64(n))
	return n, err
}

// ReaderCounting wraps r, adding the number of bytes read to n.
func ReaderCounting(r io.Reader, n *atomic.Uint64) io.Reader {
	return countingReader{Reader: r, n: n}
}

// Rewind an io.Seeker
func Rewind(o io.Seeker) error {
	_, err := o.Seek(0, io.SeekStart)
	return err
}

type writeNopCloser struct {
	io.Writer
}

func (writeNopCloser) Close() error { return nil }

// WriteNopCloser returns a WriteCloser with a no-op Close method wrapping
// the provided Writer w.
func WriteNopCloser(w io.Writer) io.WriteCloser {
	return writeNopCloser{w}
}

// Copy a file to another path
func Copy(from, to string) error {
	in, err := os.Open(from)
	if err != nil {
		return err
	}
	defer in.Close()

	i, err := in.Stat()
	if err != nil {
		return err
	}

	out, err := os.OpenFile(to, os.O_CREATE|os.O_TRUNC|os.O_WRONLY, i.Mode())
	if err != nil {
		return err
	}
	defer out.Close()

	_, err = io.Copy(out, in)
	if err != nil {
		return err
	}

	return out.Close()
}
