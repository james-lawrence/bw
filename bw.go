package bw

import (
	"crypto/md5"
	"crypto/rand"
	"encoding/base32"
	"io"
	"path/filepath"
	"time"

	"github.com/pkg/errors"
)

//go:generate protoc -I=.protocol --go_out=plugins=grpc:agent/discovery .protocol/discovery.proto
//go:generate protoc -I=.protocol --go_out=plugins=grpc:agent .protocol/agent.proto
//go:generate protoc -I=.protocol --go_out=plugins=grpc:notary .protocol/notary.proto

const (
	// DirDeploys the name of the deploys directory.
	DirDeploys = "deploys"
	// DirObservers name of the observer directory.
	DirObservers = "observers"
	// DirRaft the name of the directory dealing with the raft state.
	DirRaft = "raft"
	// DirNotary the name of the directory dealing with credentials
	DirNotary = "notary"
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
	// ArchiveFile name of the archive file stored on disk
	ArchiveFile = "archive.tar.gz"
	// DefaultRPCPort default port for RPC service.
	DefaultRPCPort = 2000
	// DefaultSWIMPort default port for peering service.
	DefaultSWIMPort = 2001
	// DefaultRaftPort default port for consensus service.
	DefaultRaftPort = 2002
	// DefaultTorrentPort default port for torrent service.
	DefaultTorrentPort = 2003
	// DefaultDiscoveryPort default port for the notary service.
	// notary service is special as its expected to be accessed without
	// a client certificate.
	DefaultDiscoveryPort = 2004
	// DefaultACMEPort port for ACME TLSALPN01 service.
	DefaultACMEPort = 2005
	// DefaultNotaryKey ...
	DefaultNotaryKey = "notary.key"
	// DefaultTLSKeyCA default name for the certificate authority key.
	DefaultTLSKeyCA = "tlsca.key"
	// DefaultTLSCertCA default name for the certificate authority certificate.
	DefaultTLSCertCA = "tlsca.cert"
)

// DeployDir return the deploy directory under the given root.
func DeployDir(root string) string {
	return filepath.Join(root, DirDeploys)
}

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
