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

// CleanSockets ...
func CleanSockets(c agent.Config) {
	logx.MaybeLog(
		errors.Wrap(
			os.RemoveAll(filepath.Join(c.Root, "bootstrap")),
			"failed to clean bootstrap service directory",
		),
	)
}

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
		socket := socket(c, fmt.Sprintf("%d.socket", i))
		if _, err := os.Stat(socket); os.IsNotExist(err) {
			return socket
		}
	}

	return ""
}

// SocketLocal generates the well known local socket from the configuration.
func SocketLocal(c agent.Config) string {
	ensureSocketDirectory(c)
	return socket(c, socketLocal)
}

// SocketQuorum generates the well known quorum socket from the configuration.
func SocketQuorum(c agent.Config) string {
	ensureSocketDirectory(c)
	return socket(c, socketQuorum)
}

func root(c agent.Config) string {
	return filepath.Join(c.Root, "bootstrap")
}

func socket(c agent.Config, socket string) string {
	return filepath.Join(root(c), socket)
}
