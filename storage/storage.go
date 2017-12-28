package storage

import (
	"io"
	"io/ioutil"

	"github.com/james-lawrence/bw/agent"
)

const (
	tmpProtocol     = "tmpfile"
	fileProtocol    = "file"
	s3Protocol      = "s3"
	torrentProtocol = "magnet"

	protocolSuffix = "://"
)

type cluster interface {
	Quorum() []agent.Peer
	Local() agent.Peer
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

func maybeIO(rc io.ReadCloser, err error) io.ReadCloser {
	if err != nil {
		return newErrReader(err)
	}

	return rc
}
