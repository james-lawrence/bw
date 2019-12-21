package acme

import (
	"context"
	"crypto/x509"
	"encoding/json"
	"io/ioutil"
	"log"
	"path/filepath"
	"sync/atomic"

	"github.com/go-acme/lego/certcrypto"
	"github.com/go-acme/lego/lego"
	"github.com/go-acme/lego/registration"
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

	return newService(c, a), nil
}

// NewService new acme service from an agent.Configuration.
func newService(c agent.Config, u account) Service {
	return Service{
		c: c,
		u: u,
		m: new(int64),
	}
}

// Service is responsible for generating and resolving ACME protocol certificates.
type Service struct {
	c agent.Config
	u account
	m *int64
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

	config := lego.NewConfig(t.u)
	config.CADirURL = t.u.CAURL
	config.Certificate.KeyType = certcrypto.RSA8192

	if client, err = lego.NewClient(config); err != nil {
		log.Println("lego client failure", err)
		return resp, status.Error(codes.Internal, "acme setup failure")
	}

	if err = autogenRegistration(t.c, client); err != nil {
		log.Println("acme registration failure", err)
		return resp, status.Error(codes.Internal, "acme setup failure")
	}

	if err = client.Challenge.SetTLSALPN01Provider(solver(t)); err != nil {
		log.Println("lego provider failure", err)
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
func (t Service) Resolution(ctx context.Context, req *ResolutionRequest) (resp *ResolutionResponse, err error) {
	return resp, status.Error(codes.Unimplemented, "resolution not yet implemented")
}

func autogenRegistration(c agent.Config, client *lego.Client) error {
	if readRegistration(c) == nil {
		_, err := genRegistration(c, client)
		return err
	}

	return nil
}

func genRegistration(c agent.Config, client *lego.Client) (zreg registration.Resource, err error) {
	var (
		encoded []byte
		reg     *registration.Resource
	)

	regp := filepath.Join(c.CredentialsDir, "acme.registration.json")

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
	regp := filepath.Join(c.CredentialsDir, "acme.registration.json")

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
