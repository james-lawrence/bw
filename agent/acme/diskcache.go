package acme

import (
	context "context"
	"crypto/md5"
	"crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"sync/atomic"
	"time"

	"cloud.google.com/go/compute/metadata"
	"github.com/go-acme/lego/v4/certcrypto"
	"github.com/go-acme/lego/v4/certificate"
	"github.com/go-acme/lego/v4/challenge"
	"github.com/go-acme/lego/v4/lego"
	"github.com/go-acme/lego/v4/providers/dns/gcloud"
	"github.com/go-acme/lego/v4/providers/dns/route53"
	"github.com/go-acme/lego/v4/registration"
	"github.com/james-lawrence/bw/agent"
	"github.com/james-lawrence/bw/certificatecache"
	"github.com/james-lawrence/bw/internal/rsax"
	"github.com/james-lawrence/bw/internal/x/logx"
	"github.com/james-lawrence/bw/internal/x/protox"
	"github.com/james-lawrence/bw/internal/x/systemx"
	"github.com/pkg/errors"
	"golang.org/x/time/rate"
	codes "google.golang.org/grpc/codes"
	status "google.golang.org/grpc/status"
	"google.golang.org/protobuf/proto"
)

// newDiskcache new acme service from an agent.Configuration.
func newDiskcache(c agent.Config, ac certificatecache.ACMEConfig, u account) DiskCache {
	return DiskCache{
		cachedir: "cached.certs",
		c:        c,
		ac:       ac,
		u:        u,
		m:        new(int64),
		rate:     rate.NewLimiter(rate.Every(ac.Rate), 1),
	}
}

// DiskCache is responsible for generating and resolving ACME protocol certificates.
type DiskCache struct {
	c        agent.Config
	ac       certificatecache.ACMEConfig
	u        account
	cachedir string
	m        *int64
	rate     *rate.Limiter
}

// Challenge initiate a challenge.
func (t DiskCache) Challenge(ctx context.Context, req *ChallengeRequest) (resp *ChallengeResponse, err error) {
	var (
		template *x509.CertificateRequest
		client   *lego.Client
		csr      []byte
		priv     []byte
		privrsa  *rsa.PrivateKey
	)

	if !atomic.CompareAndSwapInt64(t.m, 0, 1) {
		return resp, status.Error(codes.Unavailable, "challenge in progress")
	}
	defer atomic.CompareAndSwapInt64(t.m, 1, 0)

	// let's encrypt has pretty heavy rate limits in production.
	if err = t.rate.Wait(ctx); err != nil {
		log.Println("unable to clear registration", err)
		return resp, status.Error(codes.Internal, "registration reset failure")
	}

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

	if template, err = x509.ParseCertificateRequest(req.CSR); err != nil {
		log.Println("invalid certificate", err)
		return resp, status.Error(codes.FailedPrecondition, "invalid certificate request")
	}

	// Lets encrypt has rate limits on certificates generated per domain per month.
	// lets cache the generated certificates if possible.
	if resp, err = t.cachedCertificate(template); err == nil {
		return resp, nil
	}

	// cache the private key used to generate the certificate for CSR.
	if priv, err = rsax.CachedAuto(filepath.Join(t.c.Root, t.cachedir, t.digestCertificate(template)+".pem")); err != nil {
		log.Println("cache failure unable to generate or retrieve the private key for the csr", err)
		return resp, status.Error(codes.Internal, "cache failure")
	}

	if privrsa, err = rsax.Decode(priv); err != nil {
		log.Println("cache failure", err)
		return resp, status.Error(codes.Internal, "cache failure")
	}

	// BEGIN song and dance around resigning the CSR with the private key generated
	// above. this allows the certificate and key to be cached preventing rate limits
	template = &x509.CertificateRequest{
		SignatureAlgorithm: x509.SHA256WithRSA,
		Subject:            template.Subject,
		DNSNames:           template.DNSNames,
	}

	if csr, err = x509.CreateCertificateRequest(rand.Reader, template, privrsa); err != nil {
		log.Println("cache failure", err)
		return resp, status.Error(codes.Internal, "cache failure - generated csr")
	}

	if template, err = x509.ParseCertificateRequest(csr); err != nil {
		log.Println("cache failure", err)
		return resp, status.Error(codes.Internal, "cache failure - generated csr")
	}
	// END song and dance.

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

	request := certificate.ObtainForCSRRequest{
		CSR:    template,
		Bundle: true,
	}

	certificates, err := client.Certificate.ObtainForCSR(request)
	if err != nil {
		log.Println("unable to retrieve certificate", err)
		return resp, status.Error(codes.Aborted, "acme certificate signature request failed")
	}

	resp = &ChallengeResponse{
		Private:     priv,
		Certificate: certificates.Certificate,
		Authority:   certificates.IssuerCertificate,
	}

	if resp, err = t.cacheCertificate(template, resp); err != nil {
		log.Println("failed to cache challenge", err)
		return resp, status.Error(codes.Internal, "")
	}

	return resp, nil
}

func route53Provider() (p *route53.DNSProvider, err error) {
	return route53.NewDNSProvider()
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

func (t DiskCache) autoDNS() (p challenge.Provider, err error) {
	if p, err = googleProvider(); err == nil {
		return p, nil
	}

	log.Println("google dns provider failed", err)

	if p, err = route53Provider(); err == nil {
		return p, nil
	}

	log.Println("route53 dns provider failed", err)

	return nil, errors.New("unable to detect dns resolver")
}

// Resolution to a challenge.
func (t DiskCache) Resolution(ctx context.Context, req *ResolutionRequest) (resp *ResolutionResponse, err error) {
	c, err := readChallenge(t.challengeFile())
	if err != nil {
		return nil, status.Error(codes.Internal, "missing challenge")
	}
	return &ResolutionResponse{Challenge: c}, nil
}

func (t DiskCache) challengeFile() string {
	return filepath.Join(t.c.Root, "acme.challenge.proto")
}

func (t DiskCache) cachedCertificate(csr *x509.CertificateRequest) (cresp *ChallengeResponse, err error) {
	var (
		encoded []byte
	)
	cresp = &ChallengeResponse{}
	digest := t.digestCertificate(csr)
	dir := filepath.Join(t.c.Root, "cached.certs")
	defer t.clearCertCache(dir)
	path := filepath.Join(dir, fmt.Sprintf("%s.acme.certificate.proto", digest))

	if encoded, err = ioutil.ReadFile(path); err != nil {
		return cresp, err
	}

	if err = proto.Unmarshal(encoded, cresp); err != nil {
		return cresp, err
	}

	return cresp, err
}

func (t DiskCache) cacheCertificate(csr *x509.CertificateRequest, c *ChallengeResponse) (_ *ChallengeResponse, err error) {
	digest := t.digestCertificate(csr)
	dir := filepath.Join(t.c.Root, "cached.certs")
	path := filepath.Join(dir, fmt.Sprintf("%s.acme.certificate.proto", digest))

	if err = os.MkdirAll(dir, 0700); err != nil {
		return c, err
	}

	return c, protox.WriteFile(path, 0600, c)
}

func (t DiskCache) clearCertCache(dir string) {
	if err := os.MkdirAll(dir, 0700); err != nil {
		log.Println(errors.Wrap(err, "failed to create cache dir"))
		return
	}

	err := filepath.Walk(dir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		if dir == path {
			return nil
		}

		// clear cache after 8 hours
		if ctime, err := systemx.FileCreatedAt(info); err == nil && ctime.Add(20*time.Hour).Before(time.Now()) {
			logx.MaybeLog(errors.Wrap(os.RemoveAll(path), "failed to remove file"))
		} else if err != nil {
			log.Println("failed to clear cached certificates", err)
		}

		return nil
	})
	logx.MaybeLog(errors.Wrap(err, "clear cert cache failed"))
}

func (t DiskCache) digestCertificate(csr *x509.CertificateRequest) (digest string) {
	sort.Strings(csr.DNSNames)
	d := md5.Sum([]byte(csr.Subject.CommonName + strings.Join(csr.DNSNames, "")))
	return hex.EncodeToString(d[:])
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