package bw

import (
	"crypto/md5"
	"crypto/rand"
	"encoding/base32"
	"io"
	"os"
	"os/user"
	"path/filepath"
	"time"

	"github.com/james-lawrence/bw/internal/x/stringsx"
	"github.com/james-lawrence/bw/internal/x/systemx"
	"github.com/pkg/errors"
)

//go:generate protoc -I=.protocol --go_out=agent --go_opt=paths=source_relative --go-grpc_out=agent --go-grpc_opt=paths=source_relative .protocol/agent.proto
//go:generate protoc -I=.protocol --go_out=agent/discovery --go_opt=paths=source_relative --go-grpc_out=agent/discovery --go-grpc_opt=paths=source_relative .protocol/discovery.proto
//go:generate protoc -I=.protocol --go_out=agent/acme --go_opt=paths=source_relative --go-grpc_out=agent/acme --go-grpc_opt=paths=source_relative .protocol/acme.proto
//go:generate protoc -I=.protocol --go_out=notary --go_opt=paths=source_relative --go-grpc_out=notary --go-grpc_opt=paths=source_relative .protocol/notary.proto
//go:generate protoc -I=.protocol --go_out=muxer --go_opt=paths=source_relative .protocol/muxer.proto

const (
	// DirCache used as the top level cache directory below the root.
	// used to store data that can be regenerated. examples:
	// - torrents
	// - tls credentials
	// - deploy archives
	// - snapshots
	DirCache = "cache.d"
	// DirDeploys the name of the deploys directory.
	DirDeploys = "deploys"
	// DirRaft the name of the directory dealing with the raft state.
	DirRaft = "raft"
	// DirTorrents the name of the directory for storing torrent information.
	DirTorrents = "torrents"
	// DirAuthorizations the directory storing authorization credentials
	DirAuthorizations = "authorizations"
	// DirArchive the name of the directory where archives are extracted
	DirArchive = "archive"
	// EnvFile contains the filename for the deploy's environment variables.
	EnvFile = "bw.env"
	// AuthKeysFile contains the filename which holds the public keys for deployments.
	AuthKeysFile = "bw.auth.keys"
	// DefaultDeployTimeout default timeout for a deployment.
	DefaultDeployTimeout = time.Hour
	// DeployLog filename for the logs of a given deployment.
	DeployLog = "deploy.log"
	// ArchiveFile name of the archive file stored on disk
	ArchiveFile = "archive.tar.gz"
	// DefaultP2PPort port which will replace all other ports
	DefaultP2PPort = 2000
	// DefaultDirAgentCredentials ...
	DefaultDirAgentCredentials = "tls"
	// DefaultNotaryKey rsa key used by clients to identify themselves.
	DefaultNotaryKey = "private.key"
	// DefaultAgentNotaryKey rsa key used by agents to identify themselves.
	DefaultAgentNotaryKey = "p2p.pkey"
	// DefaultTLSKeyCA default name for the certificate authority key.
	DefaultTLSKeyCA = "tlsca.key"
	// DefaultTLSCertCA default name for the certificate authority certificate.
	DefaultTLSCertCA = "tlsca.cert"
)

// The various protocols defined for bearded wookie
const (
	ProtocolProxy     = "bw.proxy"
	ProtocolDiscovery = "bw.discovery"
	ProtocolSWIM      = "bw.swim"
	ProtocolRAFT      = "bw.raft"
	ProtocolAgent     = "bw.agent"
	ProtocolAutocert  = "bw.autocert"
	ProtocolTorrent   = "bw.torrent"
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

// DisplayName for the user
func DisplayName() string {
	u := systemx.CurrentUserOrDefault(user.User{Username: "unknown"})
	return stringsx.DefaultIfBlank(os.Getenv(EnvDisplayName), stringsx.DefaultIfBlank(u.Name, u.Username))
}
