// Package acme implements the acme protocol. specifically for the alpn for the cluster.
// this forces a couple requirements, the discovery service must be exposed on port 443.
// another reference implementation can be seen at:
// https://github.com/caddyserver/caddy/pull/2201/files
package acme

import (
	context "context"
	"crypto/x509"
	"log"
	"strings"
	"time"

	"github.com/hashicorp/memberlist"
	grpc "google.golang.org/grpc"
	codes "google.golang.org/grpc/codes"
	status "google.golang.org/grpc/status"

	"github.com/james-lawrence/bw"
	"github.com/james-lawrence/bw/agent"
	"github.com/james-lawrence/bw/certificatecache"
	"github.com/james-lawrence/bw/internal/envx"
	"github.com/james-lawrence/bw/internal/tlsx"
	"github.com/james-lawrence/bw/notary"
)

type rendezvous interface {
	Get([]byte) *memberlist.Node
	GetN(n int, key []byte) []*memberlist.Node
}

// ReadConfig ...
func ReadConfig(c agent.Config, path string) (svc DiskCache, err error) {
	type config struct {
		ACME certificatecache.ACMEConfig `yaml:"acme"`
	}

	var (
		cc = &config{
			ACME: certificatecache.DefaultACMEConfig(),
		}
	)

	if err = bw.ExpandAndDecodeFile(path, cc); err != nil {
		return svc, err
	}

	if nameserver := envx.String("", bw.EnvAgentACMEDNSChallengeNameServer); strings.TrimSpace(nameserver) != "" {
		cc.ACME.Challenges.NameServers = append(cc.ACME.Challenges.NameServers, nameserver)
	}

	a := account{ACMEConfig: cc.ACME, Config: c}

	return newDiskcache(c, cc.ACME, a), nil
}

type auth interface {
	Authorize(ctx context.Context) *notary.Permission
}

// NewService new acme service from an agent.Configuration.
func NewService(cache DiskCache, a auth) Server {
	return Server{
		cache: cache,
		auth:  a,
	}
}

// Server is responsible for generating and resolving ACME protocol certificates.
type Server struct {
	UnimplementedACMEServer
	cache DiskCache
	auth  auth
}

func (t Server) Bind(srv *grpc.Server) {
	RegisterACMEServer(srv, t)
}

// Challenge solve the challenge.
func (t Server) Challenge(ctx context.Context, req *CertificateRequest) (resp *CertificateResponse, err error) {
	if !t.auth.Authorize(ctx).Autocert {
		return resp, status.Error(codes.PermissionDenied, "invalid credentials")
	}

	return t.cache.Challenge(ctx, req)
}

// Resolution return the resolution
func (t Server) Resolution(ctx context.Context, req *ResolutionRequest) (resp *ResolutionResponse, err error) {
	if !t.auth.Authorize(ctx).Autocert {
		return resp, status.Error(codes.PermissionDenied, "invalid credentials")
	}

	return t.cache.Resolution(ctx, req)
}

func (t Server) Cached(ctx context.Context, req *CertificateRequest) (cached *CertificateResponse, err error) {
	var (
		csr *x509.CertificateRequest
	)

	if !t.auth.Authorize(ctx).Autocert {
		return cached, status.Error(codes.PermissionDenied, "invalid credentials")
	}

	if csr, err = x509.ParseCertificateRequest(req.CSR); err != nil {
		log.Println("invalid certificate", err)
		return cached, status.Error(codes.FailedPrecondition, "invalid certificate request")
	}

	if cached, err = t.cache.cached(csr); err != nil {
		return cached, status.Error(codes.NotFound, "cached certificate not found")
	}

	// do not return certificates nearing their expiration dates.
	// the window where we ignore the cached certificate was semi
	// arbitrarily chosen. mostly due to lets encrypts rate limits
	// of 5 / week duplicate requests.
	if cert, err := tlsx.DecodePEMCertificate(cached.Certificate); err != nil {
		log.Println("unable to parse current certificate; treating as not found", err)
		return nil, status.Error(codes.NotFound, "cached certificate not found")
	} else if cert.NotAfter.Before(time.Now().Add(72 * time.Hour)) {
		log.Printf("certificate is going to expire soon %s; treating as not found\n", cert.NotAfter)
		return nil, status.Error(codes.NotFound, "cached certificate not found")
	}

	return cached, nil
}
