// Package daemons provide simplified functions to running the various daemons
// the agent runs.
package daemons

import (
	"context"
	"net"

	"github.com/james-lawrence/bw/agent"
	"google.golang.org/grpc"
)

type cluster interface {
	Local() agent.Peer
	Peers() []agent.Peer
	Quorum() []agent.Peer
	Connect() agent.ConnectResponse
}

// Context common information passed to all daemons.
type Context struct {
	Context context.Context
}

func (t *Context) grpc(server *grpc.Server, listeners ...net.Listener) {
	for _, l := range listeners {
		go server.Serve(l)
	}
}
