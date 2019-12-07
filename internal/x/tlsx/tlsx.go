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
	"io"
	"math/big"
	"net"
	"time"

	"github.com/pkg/errors"
)

// X509Option ...
type X509Option func(*x509.Certificate)

// X509OptionSubject subject for the cert
func X509OptionSubject(s pkix.Name) X509Option {
	return func(t *x509.Certificate) {
		t.Subject = s
	}
}

// X509OptionCA enables the certificate as a ca.
func X509OptionCA() X509Option {
	return func(t *x509.Certificate) {
		t.IsCA = true
		t.KeyUsage = t.KeyUsage | x509.KeyUsageCertSign | x509.KeyUsageCRLSign
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

// X509Template ...
func X509Template(d time.Duration, options ...X509Option) (template x509.Certificate, err error) {
	var (
		serialNumber *big.Int
	)
	notBefore := time.Now()
	notAfter := notBefore.Add(d)
	serialNumberLimit := new(big.Int).Lsh(big.NewInt(1), 128)

	if serialNumber, err = rand.Int(rand.Reader, serialNumberLimit); err != nil {
		return template, errors.WithStack(err)
	}

	orgHash := md5.New()
	if _, err = io.CopyN(orgHash, rand.Reader, 1024); err != nil {
		return template, errors.WithStack(err)
	}

	template = x509.Certificate{
		SerialNumber: serialNumber,
		Subject: pkix.Name{
			Organization: []string{hex.EncodeToString(orgHash.Sum(nil))},
		},
		NotBefore:             notBefore,
		NotAfter:              notAfter,
		KeyUsage:              0,
		ExtKeyUsage:           nil,
		BasicConstraintsValid: true,
	}

	for _, opt := range options {
		opt(&template)
	}

	return template, errors.WithStack(err)
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
	if priv, err = rsa.GenerateKey(rand.Reader, bits); err != nil {
		return priv, derBytes, errors.WithStack(err)
	}

	if derBytes, err = x509.CreateCertificate(rand.Reader, &template, &template, &priv.PublicKey, priv); err != nil {
		return priv, derBytes, errors.WithStack(err)
	}

	return priv, derBytes, errors.WithStack(err)
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

// Option tls config options
type Option func(*tls.Config) error

// OptionVerifyClientIfGiven ...
func OptionVerifyClientIfGiven(c *tls.Config) error {
	c.ClientAuth = tls.VerifyClientCertIfGiven
	return nil
}
