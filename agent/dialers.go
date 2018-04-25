package agent

import (
	"log"
	"time"

	"github.com/pkg/errors"
	"google.golang.org/grpc"
	"google.golang.org/grpc/keepalive"
)

// DefaultDialerOptions sets reasonable defaults for dialing the agent.
func DefaultDialerOptions(options ...grpc.DialOption) (results []grpc.DialOption) {
	results = make([]grpc.DialOption, 0, len(options)+2)
	results = append(results, grpc.WithKeepaliveParams(
		keepalive.ClientParameters{
			Time:                5 * time.Second,
			Timeout:             15 * time.Second,
			PermitWithoutStream: true,
		},
	))
	results = append(results, grpc.WithBackoffMaxDelay(5*time.Second))

	return append(results, options...)
}

// NewDialer creates a new dialer from the provided options
func NewDialer(options ...grpc.DialOption) Dialer {
	return Dialer{
		options: options,
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
	options []grpc.DialOption
}

// Dial connects to the provided peer.
func (t Dialer) Dial(p Peer) (zeroc Client, err error) {
	var (
		addr string
	)

	if addr = RPCAddress(p); addr == "" {
		return zeroc, errors.Errorf("failed to determine address of peer: %s", p.Name)
	}

	return Dial(addr, t.options...)
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
