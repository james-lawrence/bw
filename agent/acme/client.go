package acme

import (
	"context"
	"crypto/x509"
	"log"
	"time"

	"github.com/james-lawrence/bw"
	"github.com/james-lawrence/bw/agent"
	"github.com/james-lawrence/bw/agent/dialers"
	"github.com/james-lawrence/bw/agentutil"
	"github.com/james-lawrence/bw/backoff"
	"github.com/james-lawrence/bw/internal/envx"
	"github.com/james-lawrence/bw/internal/errorsx"
	"github.com/james-lawrence/bw/internal/tlsx"
	"github.com/pkg/errors"
	"google.golang.org/grpc"
	codes "google.golang.org/grpc/codes"
	status "google.golang.org/grpc/status"
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
		backoff.Constant(15*time.Second),
		backoff.Jitter(0.25),
	)

	for i := 0; ; i++ {
		var (
			req = &CertificateRequest{
				CSR: csr,
			}
			resp *CertificateResponse
		)

		if resp, err = t.challenge(ctx, req); err == nil {
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

		delay := bo.Backoff(i).Round(50 * time.Millisecond)
		log.Println("failed to complete acme challenge", i, delay, err)

		select {
		case <-ctx.Done():
			return key, cert, authority, err
		case <-time.After(delay):
		}
	}
}

func (t Challenger) challenge(ctx context.Context, req *CertificateRequest) (resp *CertificateResponse, err error) {
	var (
		conn *grpc.ClientConn
		p    *agent.Peer
	)

	// here we select a node based on the a disciminator. that node is responsible
	// for managing the acme account key, registration, etc.
	if p, err = agent.NodeToPeer(t.rendezvous.Get([]byte(discriminator))); err != nil {
		return nil, err
	}

	if p.Name == t.local.Name {
		// before initiating a challenge check the quorum for current certificates
		// we do this because in periods when new servers are created the new node may
		// become the rendezvous node but we still have a perfectly valid certificate
		// on the current servers instead of pinging the ACME directory we instead
		// query the 2 * quorum servers requesting their cached certificate instead.
		if resp, err = t.quorumcertificate(ctx, req); err == nil {
			log.Println("quorum returned a certificate; skipping the challenge process")
			return resp, nil
		}

		if resp, err = t.cache.Challenge(ctx, req); err != nil {
			return nil, errors.Wrap(err, "disk cache")
		}

		return resp, nil
	}

	if envx.Boolean(false, bw.EnvLogsTLS, bw.EnvLogsVerbose) {
		log.Println("initiating certificate request to", agent.AutocertAddress(p))
		defer log.Println("certificate request to", agent.AutocertAddress(p), "completed")
	}

	if conn, err = dialers.NewDirect(agent.AutocertAddress(p)).DialContext(ctx, t.dialer.Defaults()...); err != nil {
		return nil, err
	}
	defer conn.Close()

	if resp, err = NewACMEClient(conn).Challenge(ctx, req); err != nil {
		return nil, err
	}

	return resp, nil
}

func (t Challenger) localcache(req *CertificateRequest, resp *CertificateResponse) (err error) {
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

// attempt to retrieve the certificate from existing quorum nodes.
func (t Challenger) quorumcertificate(ctx context.Context, req *CertificateRequest) (cached *CertificateResponse, err error) {
	var (
		conn *grpc.ClientConn
	)

	retrieve := func(ctx context.Context, p *agent.Peer) (*CertificateResponse, error) {
		address := agent.AutocertAddress(p)

		if envx.Boolean(false, bw.EnvLogsTLS, bw.EnvLogsVerbose) {
			log.Println("cached certificate request to", address, "initiated")
			defer log.Println("cached certificate request to", address, "completed")
		}

		if conn, err = dialers.NewDirect(address).DialContext(ctx, t.dialer.Defaults()...); err != nil {
			return nil, err
		}
		defer conn.Close()

		return NewACMEClient(conn).Cached(ctx, req)
	}

	for _, p := range agent.NodesToPeers(agent.LargeQuorum(t.rendezvous)...) {
		var (
			cause error
		)

		// obviously the current node that is looking for a certificate
		// will not have one available.
		if p.Name == t.local.Name {
			continue
		}

		if cached, cause = retrieve(ctx, p); cause == nil {
			ts := time.Now().Add(30 * 24 * time.Hour)
			if cert, err := tlsx.DecodePEMCertificate(cached.Certificate); err == nil {
				log.Println("certificate expires in", time.Until(cert.NotAfter), time.Until(cert.NotAfter), "<", time.Duration(req.CacheMinimumExpiration), 30*24*time.Hour)
				if cert.NotAfter.Before(ts) {
					return nil, status.Error(codes.NotFound, "certificate is expiring in 30 days ignore cache")
				}

				log.Println("cached certificate received expiration", cert.NotAfter, "<", ts, cert.NotAfter.Before(ts))
			} else {
				log.Println("unable to decode cached certificate received", err)
			}

			return cached, nil
		}

		err = errorsx.Compact(err, cause)
	}

	switch status.Code(err) {
	case codes.NotFound:
		// ignore not found case its normal and expected.
	default:
		log.Println("unable to locate certificate from quoruom", err)
	}

	return nil, status.Error(codes.NotFound, "cached certificate not found")
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

	log.Println("resolving acme challenge initiated")
	defer log.Println("resolving acme challenge completed")

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
