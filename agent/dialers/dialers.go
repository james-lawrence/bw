package dialers

import (
	"log"
	"math/rand"

	"github.com/james-lawrence/bw/agent"
	"github.com/pkg/errors"
	"google.golang.org/grpc"
)

type cluster interface {
	Quorum() []agent.Peer
}

func shuffleQuorum(q []agent.Peer) []agent.Peer {
	rand.Shuffle(len(q), func(i int, j int) {
		q[i], q[j] = q[j], q[i]
	})
	return q
}

// NewQuorum dialer.
func NewQuorum(c cluster, defaults ...grpc.DialOption) Quorum {
	return Quorum{c: c, defaults: defaults}
}

// Quorum dialers members of quorum.
type Quorum struct {
	c        cluster
	defaults []grpc.DialOption
}

// Dial with the given options
func (t Quorum) Dial(options ...grpc.DialOption) (c *grpc.ClientConn, err error) {
	err = errors.New("unable to connect")
	opts := append(t.defaults, options...)

	for _, p := range shuffleQuorum(t.c.Quorum()) {
		if c, err = grpc.Dial(agent.RPCAddress(p), opts...); err == nil {
			return c, err
		}
		log.Println("failed to connect to peer", p.Name, p.Ip)
	}

	return nil, errors.WithMessage(err, "failed to connect to a member of the quorum")
}

// NewDiscovery dials the discovery service at the provided address.
func NewDiscovery(address string, defaults ...grpc.DialOption) Discovery {
	return Discovery{
		address:  address,
		defaults: defaults,
	}
}

// Discovery ...
type Discovery struct {
	address  string
	defaults []grpc.DialOption
}

// Dial given the options
func (t Discovery) Dial(options ...grpc.DialOption) (c *grpc.ClientConn, err error) {
	return grpc.Dial(t.address, append(t.defaults, options...)...)
}
