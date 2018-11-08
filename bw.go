package bw

import (
	"crypto/md5"
	"encoding/base32"
	"io"
	"math/rand"
	"time"

	"github.com/pkg/errors"
)

//go:generate protoc -I=.protocol --go_out=plugins=grpc:agent .protocol/agent.proto
//go:generate protoc -I=.protocol --go_out=cluster .protocol/cluster.proto

const (
	// DirDeploys the name of the deploys directory.
	DirDeploys = "deploys"
	// DirObservers name of the observer directory.
	DirObservers = "observers"
	// DirRaft the name of the directory dealing with the raft state.
	DirRaft = "raft"
	// DirPlugins the name of the directory dealing with plugins for the agent.
	DirPlugins = "plugins"
	// DirTorrents the name of the directory for storing torrent information.
	DirTorrents = "torrents"
	// EnvFile environment file name.
	EnvFile = "bw.env"

	// DefaultDeployTimeout default timeout for a deployment.
	DefaultDeployTimeout = time.Hour

	// DeployLog filename for the logs of a given deployment.
	DeployLog = "deploy.log"
)

// RandomID a random identifier.
type RandomID []byte

func (t RandomID) String() string {
	return base32.StdEncoding.WithPadding(base32.NoPadding).EncodeToString(t)
}

// SimpleGenerateID ...
func SimpleGenerateID() (_ignored RandomID, err error) {
	return GenerateID(globalSrc)
}

// GenerateID generates a random ID.
func GenerateID(src *rand.Rand) (_ignored RandomID, err error) {
	const n = 1024
	randIDHash := md5.New()
	if _, err = io.CopyN(randIDHash, src, n); err != nil {
		return _ignored, errors.Wrap(err, "failed generating data for deploymentID")
	}

	return randIDHash.Sum(nil), nil
}

var globalSrc = rand.New(rand.NewSource(time.Now().Unix()))

// MustGenerateID generates a random ID, or panics.
func MustGenerateID() RandomID {
	id, err := GenerateID(globalSrc)
	if err != nil {
		panic(err)
	}

	return id
}
