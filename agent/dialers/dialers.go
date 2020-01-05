package dialers

import (
	"log"

	"github.com/hashicorp/memberlist"
	"github.com/james-lawrence/bw/agent"
	"github.com/pkg/errors"
	"google.golang.org/grpc"
)

type rendezvous interface {
	GetN(n int, key []byte) []*memberlist.Node
}

// Dialer the interface for dialing the cluster.
type Dialer interface {
	Dial(options ...grpc.DialOption) (c *grpc.ClientConn, err error)
}

// Defaults return a set of default dialing options.
// accepts additional options to merge in.
// this allows for converting from one dialer to another.
type Defaults interface {
	Defaults(combined ...grpc.DialOption) []grpc.DialOption
}

// NewQuorum dialer.
func NewQuorum(c rendezvous, defaults ...grpc.DialOption) Quorum {
	return Quorum{c: c, defaults: defaults}
}

// Quorum dialers members of quorum.
type Quorum struct {
	c        rendezvous
	defaults []grpc.DialOption
}

// Dial with the given options
func (t Quorum) Dial(options ...grpc.DialOption) (c *grpc.ClientConn, err error) {
	err = errors.New("unable to connect")
	opts := append(t.defaults, options...)

	for _, p := range agent.QuorumPeers(t.c) {
		if c, err = grpc.Dial(agent.RPCAddress(p), opts...); err == nil {
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

// NewDirect dials the provided address every time.
func NewDirect(address string, defaults ...grpc.DialOption) Direct {
	return Direct{
		address:  address,
		defaults: defaults,
	}
}

// Direct ...
type Direct struct {
	address  string
	defaults []grpc.DialOption
}

// Dial given the options
func (t Direct) Dial(options ...grpc.DialOption) (c *grpc.ClientConn, err error) {
	return grpc.Dial(t.address, append(t.defaults, options...)...)
}

// Defaults return the defaults for this dialer.
func (t Direct) Defaults(options ...grpc.DialOption) []grpc.DialOption {
	return append(t.defaults, options...)
}
