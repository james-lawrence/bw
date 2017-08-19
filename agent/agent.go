package agent

import (
	"crypto/md5"
	"hash"
	"io"
	"math/rand"
	"time"

	"google.golang.org/grpc"

	"github.com/pkg/errors"

	"bitbucket.org/jatone/bearded-wookie/deployment/agent"
)

//
// // Archive ...
// type Archive agent.Archive
//
// // Message ...
// type Message agent.Message

// GenerateID generates a random ID.
func GenerateID() (_ignored []byte, err error) {
	const n = 1024
	randIDHash := md5.New()
	if _, err = io.CopyN(randIDHash, rand.New(rand.NewSource(time.Now().Unix())), n); err != nil {
		return _ignored, errors.Wrap(err, "failed generating data for deploymentID")
	}

	return randIDHash.Sum(nil), nil
}

// RegisterServer ...
func RegisterServer(s *grpc.Server, srv agent.AgentServer) {
	agent.RegisterAgentServer(s, srv)
}

// Downloader ...
type Downloader interface {
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
