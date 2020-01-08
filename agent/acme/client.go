package acme

import (
	"context"
	"log"
	"time"

	"github.com/james-lawrence/bw/agent"
	"github.com/james-lawrence/bw/backoff"
	"github.com/james-lawrence/bw/internal/x/grpcx"
	"github.com/pkg/errors"
	"google.golang.org/grpc"
)

// this value is entirely arbitrary, because of the consistent hashing algorithms
// work we just need a constant shared value.
const discriminator = "92dcbf3f-b96c-4e97-97a3-a76dc8f1fa1e"

type dialer interface {
	Dial(agent.Peer) (agent.Client, error)
}

// Dialer for connecting to a given peer.
type _Dialer struct {
	options []grpc.DialOption
}

// Dial connects to the provided peer.
func (t _Dialer) Dial(p agent.Peer) (zeroc agent.Client, err error) {
	var (
		addr string
	)

	if addr = agent.AutocertAddress(p); addr == "" {
		return zeroc, errors.Errorf("failed to determine address of peer: %s", p.Name)
	}

	return agent.Dial(addr, t.options...)
}

// NewClient create a new Client
func NewClient(r rendezvous) Client {
	return Client{
		rendezvous: r,
		d: _Dialer{
			options: agent.DefaultDialerOptions(grpc.WithTransportCredentials(grpcx.InsecureTLS())),
		},
	}
}

// Client client to deal with acme resolutions.
type Client struct {
	rendezvous
	d dialer
}

// Challenge initiate a challenge.
func (t Client) Challenge(ctx context.Context, csr []byte) (cert []byte, authority []byte, err error) {
	bo := backoff.Jitter(time.Second, backoff.Maximum(time.Minute, backoff.Exponential(time.Second)))
	for i := 0; ; i++ {
		if cert, authority, err = t.challenge(ctx, csr); err == nil {
			return cert, authority, nil
		}

		delay := bo.Backoff(i).Round(50 * time.Millisecond)
		log.Println("failed to complete acme challenge", i, delay, err)

		select {
		case <-ctx.Done():
			return cert, authority, err
		case <-time.After(delay):
		}
	}
}

func (t Client) challenge(ctx context.Context, csr []byte) (cert []byte, authority []byte, err error) {
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
	var (
		conn *grpc.ClientConn
		p    agent.Peer
		resp *ResolutionResponse
	)

	// here we select a node based on the a disciminator. that node is responsible
	// for managing the acme account key, registration, etc.
	if p, err = agent.NodeToPeer(t.Get([]byte(discriminator))); err != nil {
		return c, err
	}

	if conn, err = agent.MaybeConn(t.d.Dial(p)); err != nil {
		return c, err
	}
	defer conn.Close()

	req := ResolutionRequest{}

	if resp, err = NewACMEClient(conn).Resolution(ctx, &req); err != nil {
		return c, err
	}

	return *resp.Challenge, err
}
