// Package acme implements the acme protocol. specifically for the alpn for the cluster.
// this forces a couple requirements, the discovery service must be exposed on port 443.
// another reference implementation can be seen at:
// https://github.com/caddyserver/caddy/pull/2201/files
package acme

import (
	"context"
	"crypto/x509"
	"encoding/json"
	"io/ioutil"
	"log"
	"os"
	"path/filepath"
	"sync/atomic"
	"errors"

	"github.com/go-acme/lego/certcrypto"
	"github.com/go-acme/lego/lego"
	"github.com/go-acme/lego/registration"
	"github.com/go-acme/lego/challenge"
	"github.com/go-acme/lego/providers/dns/gcloud"
	"cloud.google.com/go/compute/metadata"
	"github.com/hashicorp/memberlist"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	"github.com/james-lawrence/bw"
	"github.com/james-lawrence/bw/agent"
	"github.com/james-lawrence/bw/certificatecache"
	"github.com/james-lawrence/bw/internal/x/systemx"
)

type rendezvous interface {
	Get([]byte) *memberlist.Node
}

// ReadConfig ...
func ReadConfig(c agent.Config, path string) (svc Service, err error) {
	type config struct {
		ACME certificatecache.ACMEConfig `yaml:"acme"`
	}

	var (
		cc = &config{
			ACME: certificatecache.ACMEConfig{
				CAURL: lego.LEDirectoryProduction,
			},
		}
	)

	if err = bw.ExpandAndDecodeFile(path, cc); err != nil {
		return svc, err
	}

	a := account{ACMEConfig: cc.ACME, Config: c}

	return newService(c, cc.ACME, a), nil
}

// NewService new acme service from an agent.Configuration.
func newService(c agent.Config, ac certificatecache.ACMEConfig, u account) Service {
	return Service{
		c:  c,
		ac: ac,
		u:  u,
		m:  new(int64),
	}
}

// Service is responsible for generating and resolving ACME protocol certificates.
type Service struct {
	c  agent.Config
	ac certificatecache.ACMEConfig
	u  account
	m  *int64
}

// Challenge initiate a challenge.
func (t Service) Challenge(ctx context.Context, req *ChallengeRequest) (resp *ChallengeResponse, err error) {
	var (
		template *x509.CertificateRequest
		client   *lego.Client
	)

	if !atomic.CompareAndSwapInt64(t.m, 0, 1) {
		return resp, status.Error(codes.Unavailable, "challenge in progress")
	}
	defer atomic.CompareAndSwapInt64(t.m, 1, 0)

	// LEGO is retarded in its API. and we need to delete the registration so it
	// is not loaded prematurely by lego (i.e. before the new registration is generated)
	if err = clearRegistration(t.c); !os.IsNotExist(err) && err != nil {
		log.Println("unable to clear registration", err)
		return resp, status.Error(codes.Internal, "registration reset failure")
	}

	config := lego.NewConfig(t.u)
	config.CADirURL = t.u.CAURL
	config.Certificate.KeyType = certcrypto.RSA8192

	if client, err = lego.NewClient(config); err != nil {
		log.Println("lego client failure", err)
		return resp, status.Error(codes.Internal, "acme setup failure")
	}

	if _, err = genRegistration(t.c, client); err != nil {
		log.Println("acme registration failure", err)
		return resp, status.Error(codes.Internal, "acme setup failure")
	}

	if t.ac.Challenges.ALPN {
		if err = client.Challenge.SetTLSALPN01Provider(solver(t)); err != nil {
			log.Println("lego provider failure", err)
			return resp, status.Error(codes.Internal, "acme setup alpn failure")
		}
	}

	if t.ac.Challenges.DNS {
		p, err := t.autoDNS()
		if err != nil {
			log.Println("failed to detect dns provider", err)
			return resp, status.Error(codes.Internal, "acme setup dns failure")
		}

		if err = client.Challenge.SetDNS01Provider(p); err != nil {
			log.Println("lego provider failure", err)
			return resp, status.Error(codes.Internal, "acme setup dns failure")
		}
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

func googleProvider() (p *gcloud.DNSProvider, err error) {
	var (
		pid string
	)

	if pid, err = metadata.ProjectID(); err != nil {
		return nil, err
	}

	return gcloud.NewDNSProviderCredentials(pid)
}

func (t Service) autoDNS() (p challenge.Provider, err error) {
	if p, err =  googleProvider(); err == nil {
		return p, nil
	}

	log.Println("google dns provider failed", err)
	return nil, errors.New("unable to detect dns resolver")
}

// Resolution to a challenge.
func (t Service) Resolution(ctx context.Context, req *ResolutionRequest) (resp *ResolutionResponse, err error) {
	c, err := readChallenge(t.challengeFile())
	if err != nil {
		return nil, err
	}
	return &ResolutionResponse{Challenge: &c}, nil
}

func (t Service) challengeFile() string {
	return filepath.Join(t.c.Root, "acme.challenge.proto")
}

func clearRegistration(c agent.Config) (err error) {
	return os.Remove(filepath.Join(c.Root, "acme.registration.json"))
}

func genRegistration(c agent.Config, client *lego.Client) (zreg registration.Resource, err error) {
	var (
		encoded []byte
		reg     *registration.Resource
	)

	regp := filepath.Join(c.Root, "acme.registration.json")

	if reg, err = client.Registration.Register(registration.RegisterOptions{TermsOfServiceAgreed: true}); err != nil {
		return zreg, err
	}

	if encoded, err = json.Marshal(reg); err != nil {
		return zreg, err
	}

	if err = ioutil.WriteFile(regp, encoded, 0600); err != nil {
		return zreg, err
	}

	return *reg, nil
}

func readRegistration(c agent.Config) (reg *registration.Resource) {
	var (
		err     error
		encoded []byte
	)

	reg = new(registration.Resource)
	regp := filepath.Join(c.Root, "acme.registration.json")

	if !systemx.FileExists(regp) {
		return nil
	}

	if encoded, err = ioutil.ReadFile(regp); err != nil {
		log.Println("failed to read existing registration", err)
		return nil
	}

	if err = json.Unmarshal(encoded, &reg); err != nil {
		log.Println("failed to read existing registration", err)
		return nil
	}

	return reg
}
