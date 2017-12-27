package storage

import "github.com/james-lawrence/bw/agent"

const (
	tmpProtocol     = "tmpfile"
	fileProtocol    = "file"
	s3Protocol      = "s3"
	torrentProtocol = "magnet"
)

type cluster interface {
	Peers() []agent.Peer
	Local() agent.Peer
}
