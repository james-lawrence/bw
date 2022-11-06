package dialers

import (
	"context"
	"log"

	"github.com/james-lawrence/bw/agent"
	"github.com/pkg/errors"
	"google.golang.org/grpc"
)

// NewRendezvous dialer.
func NewRendezvous(key []byte, c rendezvous, defaults ...grpc.DialOption) Rendezvous {
	return Rendezvous{key: key, c: c, defaults: defaults}
}

// Rendezvous dialers a consistent member of the quorum based on a key.
type Rendezvous struct {
	key      []byte
	c        rendezvous
	defaults []grpc.DialOption
}

// Dial with given the options
func (t Rendezvous) Dial(options ...grpc.DialOption) (c *grpc.ClientConn, err error) {
	return t.DialContext(context.Background(), options...)
}

// DialContext with the given options
func (t Rendezvous) DialContext(ctx context.Context, options ...grpc.DialOption) (c *grpc.ClientConn, err error) {
	err = errors.New("unable to connect")
	opts := append(t.defaults, options...)

	for _, p := range agent.RendezvousPeers(t.key, t.c) {
		if c, err = grpc.DialContext(ctx, agent.RPCAddress(p), opts...); err == nil {
			return c, err
		}
		log.Println("failed to connect to peer", p.Name, p.Ip)
	}

	return nil, errors.WithMessage(err, "failed to connect to the quorum leader")
}

// Defaults return the defaults for this dialer.
func (t Rendezvous) Defaults(options ...grpc.DialOption) Defaulted {
	return append(t.defaults, options...)
}
