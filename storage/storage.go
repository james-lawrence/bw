// Package storage provides implementations for downloading and uploading archives
// to nodes within the cluster.
// current implementations: torrent (bittorrent), and s3.
package storage

import (
	"io"

	"github.com/james-lawrence/bw/agent"
)

const (
	s3Protocol      = "s3"
	torrentProtocol = "magnet"
)

type cluster interface {
	Quorum() []*agent.Peer
	Local() *agent.Peer
}

func newErrReader(err error) io.ReadCloser {
	return io.NopCloser(errReader{err})
}

type errReader struct {
	err error
}

func (t errReader) Read(_ []byte) (int, error) {
	return 0, t.err
}
