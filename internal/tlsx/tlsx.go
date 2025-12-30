package tlsx

import (
	"crypto/md5"
	"crypto/rand"
	"crypto/rsa"
	"crypto/tls"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/hex"
	"encoding/pem"
	"fmt"
	"io"
	"math/big"
	"net"
	"os"
	"time"

	"github.com/grantae/certinfo"
	"github.com/pkg/errors"

	"github.com/james-lawrence/bw/internal/errorsx"
)

// X509Option ...
type X509Option func(*x509.Certificate)

// X509OptionSubject subject for the cert
func X509OptionSubject(s pkix.Name) X509Option {
	return func(t *x509.Certificate) {
		t.Subject = s
	}
}

func X509OptionCA() X509Option {
	return func(t *x509.Certificate) {
		t.IsCA = true
		t.BasicConstraintsValid = true
		t.KeyUsage = t.KeyUsage | x509.KeyUsageDigitalSignature | x509.KeyUsageCertSign | x509.KeyUsageCRLSign | x509.KeyUsageKeyAgreement | x509.KeyUsageContentCommitment | x509.KeyUsageDataEncipherment | x509.KeyUsageKeyAgreement
		t.ExtKeyUsage = append(t.ExtKeyUsage, x509.ExtKeyUsageServerAuth, x509.ExtKeyUsageClientAuth)
	}
}

// X509OptionHosts set the hosts
func X509OptionHosts(names ...string) X509Option {
	return func(t *x509.Certificate) {
		for _, h := range names {
			if ip := net.ParseIP(h); ip != nil {
				t.IPAddresses = append(t.IPAddresses, ip)
			} else {
				t.DNSNames = append(t.DNSNames, h)
			}
		}
	}
}

// X509OptionUsage set the usage options for the certificate.
func X509OptionUsage(u x509.KeyUsage) X509Option {
	return func(t *x509.Certificate) {
		t.KeyUsage = t.KeyUsage | u
	}
}

// X509OptionUsageExt set the usage extension bits.
func X509OptionUsageExt(u ...x509.ExtKeyUsage) X509Option {
	return func(t *x509.Certificate) {
		t.ExtKeyUsage = u
	}
}

// X509OptionTimeWindow where the certificate is valid.
// clock can be nil.
func X509OptionTimeWindow(c clock, d time.Duration) X509Option {
	return func(cert *x509.Certificate) {
		cert.NotBefore = c.Now()
		cert.NotAfter = cert.NotBefore.Add(d)
	}
}

type clock interface {
	Now() time.Time
}

type stdlibclock struct{}

func (t stdlibclock) Now() time.Time {
	return time.Now()
}

// Default clock that can be used to generate a template cert.
func DefaultClock() clock {
	return stdlibclock{}
}

type constantclock time.Time

func (t constantclock) Now() time.Time {
	return time.Time(t)
}

func FixedClock(t time.Time) clock {
	return constantclock(t)
}

// X509TemplateRand generate a template using the provided random source.
// the clock is allowed to be nil.
func X509TemplateRand(r io.Reader, d time.Duration, c clock, options ...X509Option) (template x509.Certificate, err error) {
	var (
		serialNumber *big.Int
	)

	serialNumberLimit := new(big.Int).Lsh(big.NewInt(1), 128)

	if serialNumber, err = rand.Int(r, serialNumberLimit); err != nil {
		return template, errors.WithStack(err)
	}

	orgHash := md5.New()
	if _, err = io.CopyN(orgHash, r, 1024); err != nil {
		return template, errors.WithStack(err)
	}

	template = x509.Certificate{
		SerialNumber: serialNumber,
		Subject: pkix.Name{
			Organization: []string{hex.EncodeToString(orgHash.Sum(nil))},
		},
		KeyUsage:              0,
		ExtKeyUsage:           nil,
		BasicConstraintsValid: true,
	}

	// ensure there is always a valid window.
	X509OptionTimeWindow(c, d)(&template)

	for _, opt := range options {
		opt(&template)
	}

	return template, errors.WithStack(err)
}

// X509Template ...
func X509Template(d time.Duration, options ...X509Option) (template x509.Certificate, err error) {
	return X509TemplateRand(rand.Reader, d, stdlibclock{}, options...)
}

// SignedRSAGen ...
func SignedRSAGen(bits int, template, parent x509.Certificate, parentKey *rsa.PrivateKey) (_ *rsa.PrivateKey, derBytes []byte, err error) {
	var (
		priv *rsa.PrivateKey
	)
	if priv, err = rsa.GenerateKey(rand.Reader, bits); err != nil {
		return priv, derBytes, errors.WithStack(err)
	}

	if derBytes, err = x509.CreateCertificate(rand.Reader, &template, &parent, &priv.PublicKey, parentKey); err != nil {
		return priv, derBytes, errors.WithStack(err)
	}

	return priv, derBytes, errors.WithStack(err)
}

// SelfSignedRSAGen generate a self signed certificate.
func SelfSignedRSAGen(bits int, template x509.Certificate) (priv *rsa.PrivateKey, derBytes []byte, err error) {
	return SelfSignedRSARandGen(rand.Reader, bits, template)
}

// SelfSignedRSAGen generate a self signed certificate.
func SelfSignedRSARandGen(r io.Reader, bits int, template x509.Certificate) (priv *rsa.PrivateKey, derBytes []byte, err error) {
	if priv, err = rsa.GenerateKey(r, bits); err != nil {
		return priv, derBytes, errors.WithStack(err)
	}

	if derBytes, err = x509.CreateCertificate(r, &template, &template, &priv.PublicKey, priv); err != nil {
		return priv, derBytes, errors.WithStack(err)
	}

	return priv, derBytes, errors.WithStack(err)
}

// SelfSigned signs its own certificate ..
func SelfSigned(priv *rsa.PrivateKey, template *x509.Certificate) (_ *rsa.PrivateKey, derBytes []byte, err error) {
	return SelfSignedRand(rand.Reader, priv, template)
}

// SelfSignedRand signs its own certificate ..
func SelfSignedRand(r io.Reader, priv *rsa.PrivateKey, template *x509.Certificate) (_ *rsa.PrivateKey, derBytes []byte, err error) {
	if derBytes, err = x509.CreateCertificate(r, template, template, &priv.PublicKey, priv); err != nil {
		return priv, derBytes, errors.WithStack(err)
	}

	return priv, derBytes, nil
}

// WriteTLS ...
func WriteTLS(key *rsa.PrivateKey, derBytes []byte, err error) func(io.Writer, io.Writer, error) error {
	if err != nil {
		return func(_, _ io.Writer, _ error) error {
			return err
		}
	}

	return func(keyw, certw io.Writer, err error) error {
		if err != nil {
			return err
		}

		if err = pem.Encode(certw, &pem.Block{Type: "CERTIFICATE", Bytes: derBytes}); err != nil {
			return errors.WithStack(err)
		}

		if err = pem.Encode(keyw, &pem.Block{Type: "RSA PRIVATE KEY", Bytes: x509.MarshalPKCS1PrivateKey(key)}); err != nil {
			return errors.WithStack(err)
		}

		return nil
	}
}

// WritePrivateKey ...
func WritePrivateKey(dst io.Writer, key *rsa.PrivateKey) error {
	return errors.WithStack(pem.Encode(dst, &pem.Block{Type: "RSA PRIVATE KEY", Bytes: x509.MarshalPKCS1PrivateKey(key)}))
}

// WritePrivateKeyFile ...
func WritePrivateKeyFile(path string, key *rsa.PrivateKey) (err error) {
	var (
		dst *os.File
	)

	if dst, err = os.OpenFile(path, os.O_CREATE|os.O_TRUNC|os.O_RDWR, 0600); err != nil {
		return err
	}

	return errorsx.Compact(WritePrivateKey(dst, key), dst.Close())
}

// WriteCertificate ...
func WriteCertificate(dst io.Writer, cert []byte) error {
	return errors.WithStack(pem.Encode(dst, &pem.Block{Type: "CERTIFICATE", Bytes: cert}))
}

// WriteCertificateFile ...
func WriteCertificateFile(path string, cert []byte) (err error) {
	var (
		dst *os.File
	)

	if dst, err = os.OpenFile(path, os.O_CREATE|os.O_TRUNC|os.O_RDWR, 0600); err != nil {
		return err
	}

	return errorsx.Compact(WriteCertificate(dst, cert), dst.Sync(), dst.Close())
}

// Option tls config options
type Option func(*tls.Config) error

// OptionVerifyClientIfGiven see tls.VerifyClientCertIfGiven
func OptionVerifyClientIfGiven(c *tls.Config) error {
	c.ClientAuth = tls.VerifyClientCertIfGiven
	return nil
}

// OptionNoClientCert see tls.NoClientCert
func OptionNoClientCert(c *tls.Config) error {
	c.ClientAuth = tls.NoClientCert
	return nil
}

// OptionInsecureSkipVerify see tls.Config.InsecureSkipVerify
func OptionInsecureSkipVerify(c *tls.Config) error {
	c.InsecureSkipVerify = true
	return nil
}

// OptionNextProtocols ALPN see tls.NextProtos
func OptionNextProtocols(protocols ...string) Option {
	return func(c *tls.Config) error {
		c.NextProtos = append(c.NextProtos, protocols...)
		return nil
	}
}

// Clone ...
func Clone(c *tls.Config, options ...Option) (updated *tls.Config, err error) {
	updated = c.Clone()

	for _, opt := range options {
		if err = opt(updated); err != nil {
			return updated, err
		}
	}

	return updated, nil
}

// MustClone ...
func MustClone(c *tls.Config, options ...Option) *tls.Config {
	updated, err := Clone(c, options...)
	if err != nil {
		panic(err)
	}
	return updated
}

// PrintEncoded certificate
func PrintEncoded(cx []byte) (s string) {
	ccc, err := x509.ParseCertificates(cx)
	if err != nil {
		return fmt.Sprintf("failed: %s - %s\n", err, string(cx))
	}

	if len(ccc) == 0 {
		return fmt.Sprintf("failed no certificates found: %s", string(cx))
	}

	for _, cc := range ccc {
		ss, err := certinfo.CertificateText(cc)
		if err != nil {
			return fmt.Sprintf("failed index: %s - %s\n", err, string(cx))
		}
		s += ss + "\n"
	}

	return s
}

// PrintX509 certificate
func PrintX509(cx *x509.Certificate) (s string) {
	ss, err := certinfo.CertificateText(cx)
	if err != nil {
		return fmt.Sprintf("failed index: %s", err)
	}
	return ss
}

// Print tls certificate.
func Print(c *tls.Certificate) (s string) {
	if c == nil {
		return ""
	}

	for idx, cx := range c.Certificate {
		cc, err := x509.ParseCertificate(cx)
		if err != nil {
			s += fmt.Sprintf("failed index: %d - %s\n", idx, err)
			continue
		}

		ss, err := certinfo.CertificateText(cc)
		if err != nil {
			s += fmt.Sprintf("failed index: %d - %s\n", idx, err)
			continue
		}

		s += fmt.Sprintf("%d - %s\n", idx, ss)
	}

	return s
}

// DecodePEMCertificate decode a pem encoded x509 certiciate.
func DecodePEMCertificate(encoded []byte) (cert *x509.Certificate, err error) {
	var (
		p *pem.Block
	)

	if p, _ = pem.Decode(encoded); p == nil {
		return cert, errors.Wrap(err, "unable to decode pem certificate")
	}

	if cert, err = x509.ParseCertificate(p.Bytes); err != nil {
		return cert, errors.Wrap(err, "failed parse certificate")
	}

	return cert, nil
}

// NewDialer for tls configurations.
func NewDialer(c *tls.Config, options ...Option) *Dialer {
	return &Dialer{
		Config: MustClone(c, options...),
		NetDialer: &net.Dialer{
			Timeout: 30 * time.Second,
		},
	}
}
