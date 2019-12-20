package acme

import (
	"context"

	"github.com/james-lawrence/bw/agent"
	"google.golang.org/grpc"
)

type dialer interface {
	Dial(agent.Peer) (agent.Client, error)
}

// NewClient create a new Client
func NewClient(r rendezvous) Client {
	return Client{
		rendezvous: r,
		d: agent.NewDialer(
			agent.DefaultDialerOptions(grpc.WithInsecure())...,
		),
	}
}

// Client client to deal with acme resolutions.
type Client struct {
	rendezvous
	d dialer
}

// Challenge initiate a challenge.
func (t Client) Challenge(ctx context.Context, csr []byte) (cert []byte, authority []byte, err error) {
	// this value is entirely arbitrary, because of the consistent hashing algorithms
	// work we just need a constant shared value.
	const discriminator = "92dcbf3f-b96c-4e97-97a3-a76dc8f1fa1e"
	var (
		conn *grpc.ClientConn
		p    agent.Peer
		resp *ChallengeResponse
	)

	req := ChallengeRequest{
		CSR: csr,
	}

	// here we select a node based on the a disciminator. that node is responsible
	// for managing the acme account key, registration, etc.
	if p, err = agent.NodeToPeer(t.Get([]byte(discriminator))); err != nil {
		return cert, authority, err
	}

	if conn, err = agent.MaybeConn(t.d.Dial(p)); err != nil {
		return cert, authority, err
	}
	defer conn.Close()

	if resp, err = NewACMEClient(conn).Challenge(ctx, &req); err != nil {
		return cert, authority, err
	}

	return resp.Certificate, resp.Authority, nil
}

// Resolution retrieve a resolution.
func (t Client) Resolution(ctx context.Context) (c Challenge, err error) {
	return c, err
}
