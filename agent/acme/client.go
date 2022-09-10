package acme

import (
	"context"
	"crypto/x509"
	"log"
	"time"

	"github.com/james-lawrence/bw/agent"
	"github.com/james-lawrence/bw/agent/dialers"
	"github.com/james-lawrence/bw/agentutil"
	"github.com/james-lawrence/bw/backoff"
	"github.com/pkg/errors"
	"google.golang.org/grpc"
)

// this value is entirely arbitrary, because of how the consistent hashing algorithms
// work we just need a constant shared value.
const discriminator = "92dcbf3f-b96c-4e97-97a3-a76dc8f1fa1e"

// NewChallenger create a new Client
func NewChallenger(p *agent.Peer, r rendezvous, cache DiskCache, d dialers.Defaults) Challenger {
	qdialer := dialers.NewQuorum(
		r,
		d.Defaults()...,
	)
	return Challenger{
		local:      p,
		rendezvous: r,
		dialer:     d,
		qd:         qdialer,
		dispatcher: agentutil.NewDispatcher(qdialer),
		cache:      cache,
	}
}

// Challenger client to deal with acme challenges.
type Challenger struct {
	local *agent.Peer
	rendezvous
	dialer     dialers.Defaults
	qd         dialers.Quorum
	dispatcher agent.Dispatcher
	cache      DiskCache
}

// Challenge initiate a challenge.
func (t Challenger) Challenge(ctx context.Context, csr []byte) (key, cert, authority []byte, err error) {
	bo := backoff.New(
		backoff.Exponential(time.Second),
		backoff.Maximum(time.Minute),
		backoff.Jitter(0.25),
	)

	for i := 0; ; i++ {
		if key, cert, authority, err = t.challenge(ctx, csr); err == nil {
			return key, cert, authority, nil
		}

		delay := bo.Backoff(i).Round(50 * time.Millisecond)
		log.Println("failed to complete acme challenge", i, delay, err)

		select {
		case <-ctx.Done():
			return key, cert, authority, err
		case <-time.After(delay):
		}
	}
}

func (t Challenger) challenge(ctx context.Context, csr []byte) (key, cert, authority []byte, err error) {
	var (
		conn *grpc.ClientConn
		p    *agent.Peer
		resp *ChallengeResponse
	)

	req := &ChallengeRequest{
		CSR: csr,
	}

	// here we select a node based on the a disciminator. that node is responsible
	// for managing the acme account key, registration, etc.
	if p, err = agent.NodeToPeer(t.rendezvous.Get([]byte(discriminator))); err != nil {
		return key, cert, authority, err
	}

	if p.Name == t.local.Name {
		if resp, err = t.cache.Challenge(ctx, req); err != nil {
			return nil, nil, nil, errors.Wrap(err, "disk cache")
		}

		// dispatch the fact we've resolved the challege to the quorum.
		// its perfectly okay for this to fail. its just another cache.
		evt := agentutil.TLSEventMessage(
			t.local,
			resp.Authority,
			resp.Private,
			resp.Certificate,
		)
		if err = t.dispatcher.Dispatch(ctx, evt); err != nil {
			log.Println("unable to dispatch tls event to quorum", err)
		}

		return resp.Private, resp.Certificate, resp.Authority, nil
	}

	// log.Println("attempting to obtain tls from quorum")

	log.Println("obtaining from", agent.AutocertAddress(p))
	if conn, err = dialers.NewDirect(agent.AutocertAddress(p)).DialContext(ctx, t.dialer.Defaults()...); err != nil {
		return key, cert, authority, err
	}
	defer conn.Close()

	if resp, err = NewACMEClient(conn).Challenge(ctx, req); err != nil {
		return key, cert, authority, err
	}

	// attempt to cache the certificate locally.
	// we do this because as the cluster mutates over time
	// with new servers the server responsible for dealing with resolutions
	// can change and we want to limit the number of requests for certificates.
	// by caching here we ensure that if this server becomes responsible for issuing
	// certs we'll have the certificate ready to go.
	if err = t.localcache(req, resp); err != nil {
		log.Println("failed to cache certificate locally", err)
	}

	return resp.Private, resp.Certificate, resp.Authority, nil
}

func (t Challenger) localcache(req *ChallengeRequest, resp *ChallengeResponse) (err error) {
	var (
		template *x509.CertificateRequest
	)

	if template, err = x509.ParseCertificateRequest(req.CSR); err != nil {
		return errors.Wrap(err, "failed to parse CSR for caching")
	}

	if _, err = t.cache.cacheCertificate(template, resp); err != nil {
		return err
	}

	return nil
}

// NewResolver create a new Client
func NewResolver(p *agent.Peer, r rendezvous, cache DiskCache, d dialers.Defaults) Resolver {
	return Resolver{
		local:      p,
		rendezvous: r,
		dialer:     d,
		cache:      cache,
	}
}

// Client client to deal with acme resolutions.
type Resolver struct {
	local      *agent.Peer
	rendezvous rendezvous
	dialer     dialers.Defaults
	cache      DiskCache
}

// Resolution retrieve a resolution.
func (t Resolver) Resolution(ctx context.Context) (c *Challenge, err error) {
	var (
		conn *grpc.ClientConn
		p    *agent.Peer
		resp *ResolutionResponse
		req  = &ResolutionRequest{}
	)

	// here we select a node based on the a disciminator. that node is responsible
	// for managing the acme account key, registration, etc.
	if p, err = agent.NodeToPeer(t.rendezvous.Get([]byte(discriminator))); err != nil {
		return c, err
	}

	if p.Name == t.local.Name {
		if resp, err = t.cache.Resolution(ctx, req); err != nil {
			return nil, errors.Wrap(err, "disk cache")
		}
		return resp.Challenge, nil
	}

	if conn, err = dialers.NewDirect(agent.AutocertAddress(p)).DialContext(ctx, t.dialer.Defaults()...); err != nil {
		return c, err
	}
	defer conn.Close()

	if resp, err = NewACMEClient(conn).Resolution(ctx, req); err != nil {
		return c, err
	}

	return resp.Challenge, err
}
