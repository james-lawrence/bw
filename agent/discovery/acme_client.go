package discovery

import (
	"context"

	"github.com/james-lawrence/bw/agent"
	"google.golang.org/grpc"
)

// NewACMEConn create a new ACMEConn
func NewACMEConn(r rendezvous, d agent.Dialer) ACMEConn {
	return ACMEConn{
		rendezvous: r,
		d:          d,
	}
}

// ACMEConn client to deal with acme resolutions.
type ACMEConn struct {
	rendezvous
	d agent.Dialer
}

// Challenge initiate a challenge.
func (t ACMEConn) Challenge(ctx context.Context, req ChallengeRequest) (_ *ChallengeResponse, err error) {
	var (
		conn *grpc.ClientConn
		p    agent.Peer
	)

	// here we select a node based on the domain we are requesting. that node is responsible
	// for managing the acme account key, registration, etc.
	if p, err = agent.NodeToPeer(t.Get([]byte(req.Domain))); err != nil {
		return nil, err
	}

	if conn, err = agent.MaybeConn(t.d.Dial(p)); err != nil {
		return nil, err
	}

	return NewACMEClient(conn).Challenge(ctx, &req)
}

// Resolution retrieve a resolution.
func (t ACMEConn) Resolution(ctx context.Context) (c Challenge, err error) {
	return c, err
}
