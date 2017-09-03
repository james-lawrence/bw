package inflaters

import (
	"io"
	"log"
	"os"

	"github.com/pkg/errors"
)

const (
	fileProtocol = "file://"
	// s3Protocol   = "s3://"
	// gitProtocol  = "git://"
)

// Inflater takes a reader containing an archive and inflates it.
type Inflater interface {
	Inflate(io.Reader) error
}

// New creates a new inflater based on the source uri and destination names.
// In the future it may fall back to a temp file + looking for magic values as a
// last resort.
func New(location, destination string, perm os.FileMode) Inflater {
	return Copy{
		FileMode:    perm,
		Destination: destination,
	}
}

// Copy - inflaters by form of a simple copy from the source reader.
type Copy struct {
	FileMode    os.FileMode
	Destination string
}

// Inflate implements the Inflater interface.
func (t Copy) Inflate(r io.Reader) (err error) {
	var (
		dst *os.File
	)

	if dst, err = os.OpenFile(t.Destination, os.O_CREATE|os.O_TRUNC|os.O_RDWR, t.FileMode); err != nil {
		return errors.WithStack(err)
	}
	defer dst.Close()

	if _, err = io.Copy(dst, r); err != nil {
		log.Println("copy failed removing created file")
		log.Println("failed to remove copy:", os.Remove(t.Destination))
		return errors.WithStack(err)
	}

	return nil
}
