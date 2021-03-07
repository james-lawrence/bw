package dialers

import (
	"context"
	"log"

	"github.com/james-lawrence/bw/agent"
	"github.com/pkg/errors"
	"google.golang.org/grpc"
)

// NewQuorum dialer.
func NewQuorum(c rendezvous, defaults ...grpc.DialOption) Quorum {
	return Quorum{c: c, defaults: defaults}
}

// Quorum dialers members of quorum.
type Quorum struct {
	c        rendezvous
	defaults []grpc.DialOption
}

// Dial with given the options
func (t Quorum) Dial(options ...grpc.DialOption) (c *grpc.ClientConn, err error) {
	return t.DialContext(context.Background(), options...)
}

// DialContext with the given options
func (t Quorum) DialContext(ctx context.Context, options ...grpc.DialOption) (c *grpc.ClientConn, err error) {
	err = errors.New("unable to connect")
	opts := append(t.defaults, options...)

	for _, p := range agent.QuorumPeers(t.c) {
		if c, err = grpc.DialContext(ctx, agent.RPCAddress(p), opts...); err == nil {
			return c, err
		}
		log.Println("failed to connect to peer", p.Name, p.Ip)
	}

	return nil, errors.WithMessage(err, "failed to connect to a member of the quorum")
}

// Defaults return the defaults for this dialer.
func (t Quorum) Defaults(options ...grpc.DialOption) []grpc.DialOption {
	return append(t.defaults, options...)
}
