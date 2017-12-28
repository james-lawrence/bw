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
	// DirRaft the name of the directory dealing with the raft state.
	DirRaft = "raft"
	// DirPlugins the name of the directory dealing with plugins for the agent.
	DirPlugins = "plugins"
	// DirTorrents the name of the directory for storing torrent information.
	DirTorrents = "torrents"
)

// RandomID a random identifier.
type RandomID []byte

func (t RandomID) String() string {
	return base32.StdEncoding.WithPadding(base32.NoPadding).EncodeToString(t)
}

// GenerateID generates a random ID.
func GenerateID() (_ignored RandomID, err error) {
	const n = 1024
	randIDHash := md5.New()
	if _, err = io.CopyN(randIDHash, rand.New(rand.NewSource(time.Now().Unix())), n); err != nil {
		return _ignored, errors.Wrap(err, "failed generating data for deploymentID")
	}

	return randIDHash.Sum(nil), nil
}

// MustGenerateID generates a random ID, or panics.
func MustGenerateID() RandomID {
	id, err := GenerateID()
	if err != nil {
		panic(err)
	}

	return id
}
