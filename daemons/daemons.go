// Package daemons provide simplified functions to running the various daemons
// the agent runs. include initialization and setup utility functions.
package daemons

import (
	"context"
	"crypto/tls"
	"log"
	"net"
	"sync"

	"github.com/james-lawrence/bw/agent"
	"github.com/james-lawrence/bw/agent/acme"
	"github.com/james-lawrence/bw/agent/dialers"
	_cluster "github.com/james-lawrence/bw/cluster"
	"github.com/james-lawrence/bw/clustering"
	"github.com/james-lawrence/bw/clustering/raftutil"
	"github.com/james-lawrence/bw/deployment"
	"github.com/james-lawrence/bw/internal/x/errorsx"
	"github.com/james-lawrence/bw/muxer"
	"github.com/james-lawrence/bw/notary"
	"github.com/pkg/errors"

	"github.com/hashicorp/memberlist"
	"google.golang.org/grpc"
	"google.golang.org/grpc/keepalive"
)

type cluster interface {
	Local() *agent.Peer
	Peers() []*agent.Peer
	Quorum() []*agent.Peer
	Connect() agent.ConnectResponse
	// todo reduce number of methods.
	Members() []*memberlist.Node
	Get([]byte) *memberlist.Node
	GetN(n int, key []byte) []*memberlist.Node
	LocalNode() *memberlist.Node
}

type dialer interface {
	DialContext(ctx context.Context, network string, address string) (net.Conn, error)
}

// Context common information passed to all daemons.
type Context struct {
	Local             *_cluster.Local
	Listener          net.Listener
	Dialer            dialers.Defaults
	Muxer             *muxer.M
	ConfigurationFile string
	Config            agent.Config
	Context           context.Context
	Shutdown          context.CancelFunc
	Cleanup           *sync.WaitGroup
	NotaryStorage     notary.Composite
	NotaryAuth        notary.Auth
	Raft              raftutil.Protocol
	Cluster           cluster
	Bootstrapper      clustering.Joiner
	RPCCredentials    *tls.Config
	RPCKeepalive      keepalive.ServerParameters
	PeeringEvents     *_cluster.EventsQueue
	Results           chan *deployment.DeployResult
	Debug             bool
	DebugLog          *log.Logger
	ACMECache         acme.DiskCache
	Inmem             *grpc.ClientConn
	P2PPublicKey      []byte
}

// MuxerListen ...
func (t *Context) MuxerListen(ctx context.Context, listeners ...net.Listener) {
	t.shutdown("muxer", listeners...)
	log.Println("establishing muxer")
	for _, l := range listeners {
		log.Println("listening at", l.Addr().String())
		go func(l net.Listener) {
			muxer.Listen(ctx, t.Muxer, l)
		}(l)
	}
}
func (t *Context) grpc(name string, server *grpc.Server, listeners ...net.Listener) {
	t.shutdown(name, listeners...)
	log.Println("listening", name)
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
