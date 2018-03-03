package agent

import (
	"log"

	"github.com/pkg/errors"
	"google.golang.org/grpc"
)

// NewDialer creates a new dialer from the provided options
func NewDialer(transportOpts ...grpc.DialOption) Dialer {
	return Dialer{
		transportOptions: transportOpts,
	}
}

// NewQuorumDialer creates a new dialer that connects to a member of the quorum.
func NewQuorumDialer(d Dialer) QuorumDialer {
	return QuorumDialer{
		dialer: d,
	}
}

// Dialer interface for connecting to a given peer.
type Dialer struct {
	transportOptions []grpc.DialOption
}

// Dial connects to the provided peer.
func (t Dialer) Dial(p Peer) (zeroc Client, err error) {
	var (
		addr string
	)

	if addr = RPCAddress(p); addr == "" {
		return zeroc, errors.Errorf("failed to determine address of peer: %s", p.Name)
	}

	return Dial(addr, t.transportOptions...)
}

// QuorumDialer connects to a member of the quorum.
type QuorumDialer struct {
	dialer Dialer
}

// Dial connects to a member of the quorum based on the cluster.
func (t QuorumDialer) Dial(c cluster) (client Client, err error) {
	for _, p := range c.Quorum() {
		if client, err = t.dialer.Dial(p); err == nil {
			break
		}
		log.Println("failed to connect to peer", p.Name, p.Ip)
	}

	return client, errors.WithMessage(err, "failed to connect to a member of the quorum")
}
