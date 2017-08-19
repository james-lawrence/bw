package agent

import (
	"io"
	"io/ioutil"
	"log"
	"os"
	"strings"

	"github.com/pkg/errors"
)

const (
	fileProtocol = "file://"
	// s3Protocol   = "s3://"
)

// NewDownloader ...
func NewDownloader(location string) Downloader {

	switch {
	case strings.HasPrefix(location, fileProtocol):
		return DownloadFile{Path: strings.TrimPrefix(location, fileProtocol)}
	default:
		return downloader{newErrReader(errors.Errorf("unknown archive protocol: %s", location))}
	}
}

func newErrReader(err error) io.ReadCloser {
	return ioutil.NopCloser(errReader{err})
}

type errReader struct {
	err error
}

func (t errReader) Read(_ []byte) (int, error) {
	return 0, t.err
}

type downloader struct {
	io.ReadCloser
}

func (t downloader) Download() io.ReadCloser {
	return t.ReadCloser
}

// DownloadFile ...
type DownloadFile struct {
	Path string
}

// Download ...
func (t DownloadFile) Download() (src io.ReadCloser) {
	var (
		err error
	)
	if src, err = os.Open(t.Path); err != nil {
		return newErrReader(errors.Wrapf(err, "failed to open file: %s", t.Path))
	}
	log.Println("created file reader", t.Path)
	return src
}
