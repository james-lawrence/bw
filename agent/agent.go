package agent

import (
	"hash"
	"io"

	"google.golang.org/grpc"

	"bitbucket.org/jatone/bearded-wookie/deployment/agent"
)

//
// // Archive ...
// type Archive agent.Archive
//
// // Message ...
// type Message agent.Message

// RegisterServer ...
func RegisterServer(s *grpc.Server, srv agent.AgentServer) {
	agent.RegisterAgentServer(s, srv)
}

// downloader ...
type downloader interface {
	Download() io.ReadCloser
}

// Uploader ...
type Uploader interface {
	Upload(io.Reader) (hash.Hash, error)
	Info() (hash.Hash, string, error)
}

// Eventer ...
type Eventer interface {
	Send(...agent.Message)
}
