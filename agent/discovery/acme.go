package discovery

import (
	"context"
	"crypto/x509"
	"log"
	"sync/atomic"
	"time"

	"github.com/go-acme/lego/certcrypto"
	"github.com/go-acme/lego/lego"
	"github.com/go-acme/lego/registration"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	"github.com/james-lawrence/bw/agent"
)

// NewACMEService new acme service from an agent.Configuration.
func NewACMEService(c agent.Config, u registration.User) ACMEService {
	return ACMEService{
		c: c,
		u: u,
		m: new(int64),
	}
}

type solver ACMEService

func (t solver) Present(domain, token, keyAuth string) error {
	log.Println("PRESENT INITIATED", domain, token, keyAuth)
	defer log.Println("PRESENT COMPLETED", domain, token, keyAuth)
	time.Sleep(time.Second)
	return nil
}

func (t solver) CleanUp(domain, token, keyAuth string) error {
	log.Println("CLEANUP INTIATED", domain, token, keyAuth)
	defer log.Println("CLEANUP COMPLETED", domain, token, keyAuth)

	time.Sleep(5 * time.Second)
	return nil
}

// ACMEService is responsible for generating and resolving ACME protocol certificates.
type ACMEService struct {
	c agent.Config
	u registration.User
	m *int64
}

// Challenge initiate a challenge.
func (t ACMEService) Challenge(ctx context.Context, req *ChallengeRequest) (resp *ChallengeResponse, err error) {
	var (
		template *x509.CertificateRequest
		client   *lego.Client
	)

	if !atomic.CompareAndSwapInt64(t.m, 0, 1) {
		return resp, status.Error(codes.Unavailable, "challenge in progress")
	}
	defer atomic.CompareAndSwapInt64(t.m, 1, 0)

	config := lego.NewConfig(t.u)
	// config.CADirURL = t.Config.CAURL
	config.Certificate.KeyType = certcrypto.RSA8192

	if client, err = lego.NewClient(config); err != nil {
		return resp, status.Error(codes.Internal, "acme setup failure")
	}

	if err = client.Challenge.SetTLSALPN01Provider(solver(t)); err != nil {
		return resp, status.Error(codes.Internal, "acme setup failure")
	}

	if template, err = x509.ParseCertificateRequest(req.CSR); err != nil {
		log.Println("invalid certificate", err)
		return resp, status.Error(codes.FailedPrecondition, "invalid certificate request")
	}

	certificates, err := client.Certificate.ObtainForCSR(*template, true)
	if err != nil {
		log.Println("unable to retrieve certificate", err)
		return resp, status.Error(codes.Aborted, "acme certificate signature request failed")
	}

	return &ChallengeResponse{
		Certificate: certificates.Certificate,
		Authority:   certificates.IssuerCertificate,
	}, nil
}

// Resolution to a challenge.
func (t ACMEService) Resolution(ctx context.Context, req *ResolutionRequest) (resp *ResolutionResponse, err error) {
	return resp, status.Error(codes.Unimplemented, "resolution not yet implemented")
}
