package bw

import (
	"crypto/md5"
	"crypto/rand"
	"encoding/base32"
	"io"
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
	// DefaultRPCPort default port for RPC service.
	DefaultRPCPort = 2000
	// DefaultSWIMPort default port for peering service.
	DefaultSWIMPort = 2001
	// DefaultRaftPort default port for consensus service.
	DefaultRaftPort = 2002
	// DefaultTorrentPort default port for torrent service.
	DefaultTorrentPort = 2003
	// DefaultACMEPort port for ACME TLSALPN01 service.
	DefaultACMEPort = 2004
)

// RandomID a random identifier.
type RandomID []byte

func (t RandomID) String() string {
	return base32.StdEncoding.WithPadding(base32.NoPadding).EncodeToString(t)
}

// SimpleGenerateID ...
func SimpleGenerateID() (_ignored RandomID, err error) {
	return GenerateID(rand.Reader)
}

// GenerateID generates a random ID.
func GenerateID(src io.Reader) (_ignored RandomID, err error) {
	const n = 1024
	var (
		written int64
	)

	randIDHash := md5.New()
	if written, err = io.CopyN(randIDHash, src, n); err != nil {
		return _ignored, errors.Wrap(err, "failed generating data for deploymentID")
	}

	if written != n {
		return _ignored, errors.Errorf("didn't read enough data: wanted %d, read %d", n, written)
	}

	return randIDHash.Sum(nil), nil
}

// MustGenerateID generates a random ID, or panics.
func MustGenerateID() RandomID {
	id, err := GenerateID(rand.Reader)
	if err != nil {
		panic(err)
	}

	return id
}
