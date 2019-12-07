// Package daemons provide simplified functions to running the various daemons
// the agent runs. include initialization and setup utility functions.
package daemons

import (
	"context"
	"log"
	"net"
	"sync"

	"github.com/james-lawrence/bw/agent"
	"github.com/james-lawrence/bw/clustering/raftutil"
	"github.com/james-lawrence/bw/deployment"
	"github.com/james-lawrence/bw/internal/x/errorsx"
	"github.com/james-lawrence/bw/storage"
	"github.com/pkg/errors"

	"github.com/hashicorp/memberlist"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials"
)

type cluster interface {
	Local() agent.Peer
	Peers() []agent.Peer
	Quorum() []agent.Peer
	Connect() agent.ConnectResponse
	// todo reduce number of methods.
	Members() []*memberlist.Node
	Get([]byte) *memberlist.Node
	GetN(n int, key []byte) []*memberlist.Node
	LocalNode() *memberlist.Node
}

type dialer interface {
	Dial(p agent.Peer) (zeroc agent.Client, err error)
}

// Context common information passed to all daemons.
type Context struct {
	Context        context.Context
	Shutdown       context.CancelFunc
	Cleanup        *sync.WaitGroup
	Upload         storage.UploadProtocol
	Download       storage.DownloadProtocol
	Raft           raftutil.Protocol
	RPCCredentials credentials.TransportCredentials
	Results        chan deployment.DeployResult
}

func (t *Context) grpc(name string, server *grpc.Server, listeners ...net.Listener) {
	t.shutdown(name, listeners...)

	for _, l := range listeners {
		go server.Serve(l)
	}
}

func (t *Context) shutdown(name string, listeners ...net.Listener) {
	t.Cleanup.Add(1)
	go func() {
		defer t.Cleanup.Done()
		<-t.Context.Done()
		for _, l := range listeners {
			log.Println(name, "shutdown", errorsx.Compact(errors.Wrap(l.Close(), "failed"), errorsx.String("complete")))
		}
	}()
}
