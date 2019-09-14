package bootstrap

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/james-lawrence/bw/agent"
	"github.com/james-lawrence/bw/internal/x/logx"
	"github.com/pkg/errors"
)

// well known bootstrap sockets
const (
	socketLocal  = "local.socket"
	socketQuorum = "quorum.socket"
)

func ensureSocketDirectory(c agent.Config) {
	if err := os.MkdirAll(filepath.Join(c.Root, "bootstrap"), 0744); err != nil {
		logx.MaybeLog(errors.Wrap(err, "failed to create bootstrap socket directory"))
	}
}

// SocketAuto increments starting at 0 until it finds a available socket.
// allows upwards of 10 potential bootstrap services, this limit is arbitrary.
func SocketAuto(c agent.Config) string {
	ensureSocketDirectory(c)

	for i := 0; i < 10; i++ {
		socket := filepath.Join(c.Root, "bootstrap", fmt.Sprintf("%d.socket", i))
		if _, err := os.Stat(socket); os.IsNotExist(err) {
			return socket
		}
	}

	return ""
}

// SocketLocal generates the well known local socket from the configuration.
func SocketLocal(c agent.Config) string {
	ensureSocketDirectory(c)
	return filepath.Join(c.Root, "bootstrap", socketLocal)
}

// SocketQuorum generates the well known quorum socket from the configuration.
func SocketQuorum(c agent.Config) string {
	ensureSocketDirectory(c)
	return filepath.Join(c.Root, "bootstrap", socketQuorum)
}
