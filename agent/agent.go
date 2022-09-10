package agent

import (
	"context"
	"hash"
	"io"
	"net"

	"google.golang.org/grpc"
)

// QuorumKey used for determining possible candidates for the quorum nodes
// within the cluster.
const (
	QuorumMaximum = 5
	QuorumDefault = 3
	QuorumKey     = "leaders"
)

// Dispatcher - interface for dispatching messages.
type Dispatcher interface {
	Dispatch(context.Context, ...*Message) error
}

// ConnectableDispatcher ...
type ConnectableDispatcher interface {
	Dispatcher
	Connect(chan *Message) (net.Listener, *grpc.Server, error)
}

// Client - client facade interface.
type Client interface {
	Conn() *grpc.ClientConn
	Close() error
	Shutdown(ctx context.Context) error
	Upload(ctx context.Context, initiator string, srcbytes uint64, src io.Reader) (*Archive, error)
	RemoteDeploy(ctx context.Context, dopts *DeployOptions, a *Archive, peers ...*Peer) error
	Deploy(context.Context, *DeployOptions, *Archive) (*Deploy, error)
	Connect(ctx context.Context) (*ConnectResponse, error)
	Cancel(context.Context, *CancelRequest) error
	NodeCancel(ctx context.Context) error
	QuorumInfo(ctx context.Context) (*InfoResponse, error)
	Info(ctx context.Context) (*StatusResponse, error)
	Watch(ctx context.Context, out chan<- *Message) error
	Dispatch(ctx context.Context, messages ...*Message) error
	Logs(context.Context, *Peer, []byte) io.ReadCloser
}

// DeployClient - facade interface.
type DeployClient interface {
	Close() error
	Upload(ctx context.Context, initiator string, srcbytes uint64, src io.Reader) (*Archive, error)
	RemoteDeploy(ctx context.Context, dopts *DeployOptions, a *Archive, peers ...*Peer) error
	Watch(ctx context.Context, out chan<- *Message) error
	Logs(context.Context, *Peer, []byte) io.ReadCloser
	Cancel(context.Context, *CancelRequest) error
}

type cluster interface {
	Local() *Peer
	Peers() []*Peer
	Quorum() []*Peer
}

// Uploader ...
type Uploader interface {
	Upload(io.Reader) (hash.Hash, error)
	Info() (hash.Hash, string, error)
}

// DetectQuorum detects a peer based on the compare function.
// from the set of quorum nodes.
func DetectQuorum(c cluster, compare func(*Peer) bool) *Peer {
	for _, n := range c.Quorum() {
		if compare(n) {
			return n
		}
	}

	return nil
}

// IsLeader compares the address of a peer to the provided leader address.
func IsLeader(address string) func(*Peer) bool {
	return func(n *Peer) bool {
		return RaftAddress(n) == address
	}
}

// IsInQuorum ...
func IsInQuorum(p *Peer) func(*Peer) bool {
	return func(n *Peer) bool {
		return n.Name == p.Name
	}
}
