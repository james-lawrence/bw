package certificatecache

import (
	"context"
	"crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/asn1"
	"log"
	"os"
	"path/filepath"
	"time"

	"github.com/go-acme/lego/v4/lego"
	"github.com/pkg/errors"

	"github.com/james-lawrence/bw/internal/errorsx"
	"github.com/james-lawrence/bw/internal/rsax"
)

// export LEGO_CA_CERTIFICATES="${HOME}/go/src/github.com/letsencrypt/pebble/test/certs/pebble.minica.pem"
// cd ${HOME}/go/src/github.com/letsencrypt/pebble; pebble -config ./test/config/pebble-config.json

// DefaultACMEConfig defines the default configuration for the ACME
// protocol.
// Let's encrypt has some pretty heavy restrictions for rate limiting.
// 5 per hour. so we'll rate limit to 13 minutes by default.
func DefaultACMEConfig() ACMEConfig {
	return ACMEConfig{
		Rate:  15 * time.Minute,
		CAURL: lego.LEDirectoryProduction,
		Challenges: challenges{
			ALPN: true,
		},
	}
}

type challenges struct {
	DNS  bool
	ALPN bool
}

// ACMEConfig configuration for ACME credentials
type ACMEConfig struct {
	Rate               time.Duration `yaml:"frequency"` // frequency of attempts.
	Challenges         challenges    `yaml:"challenges"`
	CAURL              string        `yaml:"caurl"`
	Email              string        `yaml:"email"`
	Network            string        `yaml:"network"`
	Country            []string      `yaml:"country"`  // Country Codes for the CSR
	Province           []string      `yaml:"province"` // Provinces for the CSR
	Locality           []string      `yaml:"locality"`
	Organization       []string      `yaml:"organization"`
	OrganizationalUnit []string      `yaml:"organizationalUnit"`
	DNSNames           []string      `yaml:"dns"`    // alternative dns names
	Secret             string        `yaml:"secret"` // secret for generating account key.
}

type challenger interface {
	Challenge(ctx context.Context, csr []byte) (key, cert, authority []byte, err error)
}

// NewACME certificate refresh.
func NewACME(dir string, a challenger) ACME {
	return ACME{
		CertificateDir: dir,
		Config:         DefaultACMEConfig(),
		c:              a,
	}
}

// ACME provides the ability to generate certificates using the acme protocol.
type ACME struct {
	c              challenger
	CertificateDir string     `yaml:"credentialsDir"`
	CommonName     string     `yaml:"servername"` // common name for certificate, usually a domain name. pulls from the servername of the configuration.
	Config         ACMEConfig `yaml:"acme"`
}

// Refresh the credentials if necessary.
func (t ACME) Refresh() (err error) {
	var (
		key       []byte
		cert      []byte
		authority []byte
		priv      *rsa.PrivateKey
	)

	if priv, err = rsax.MaybeDecode(rsax.CachedAuto(filepath.Join(t.CertificateDir, DefaultTLSKeyServer))); err != nil {
		return err
	}

	subj := pkix.Name{
		CommonName:         t.CommonName,
		Country:            t.Config.Country,
		Province:           t.Config.Province,
		Locality:           t.Config.Locality,
		Organization:       t.Config.Organization,
		OrganizationalUnit: t.Config.OrganizationalUnit,
		ExtraNames: []pkix.AttributeTypeAndValue{
			{
				Type: asn1.ObjectIdentifier{1, 2, 840, 113549, 1, 9, 1},
				Value: asn1.RawValue{
					Tag:   asn1.TagIA5String,
					Bytes: []byte(t.Config.Email),
				},
			},
		},
	}

	template := &x509.CertificateRequest{
		Subject:            subj,
		SignatureAlgorithm: x509.SHA256WithRSA,
		DNSNames:           append(t.Config.DNSNames, t.CommonName),
	}

	csr, err := x509.CreateCertificateRequest(rand.Reader, template, priv)
	if err != nil {
		return errors.Wrap(err, "failed to create CSR")
	}

	log.Println("certificate request initiated")
	if key, cert, authority, err = t.c.Challenge(context.Background(), csr); err != nil {
		return errors.Wrap(err, "failed to obtain certificates")
	}
	log.Println("certificate request completed")

	capath := filepath.Join(t.CertificateDir, DefaultTLSCertCA)
	certpath := filepath.Join(t.CertificateDir, DefaultTLSCertServer)
	keypath := filepath.Join(t.CertificateDir, DefaultTLSKeyServer)

	log.Println("writing authority certificate", capath)
	if err = os.WriteFile(capath, authority, 0600); err != nil {
		return errorsx.MaybeLog(errors.Wrapf(err, "failed to write certificate authority to %s", capath))
	}

	log.Println("writing certificate", certpath)
	if err = os.WriteFile(certpath, cert, 0600); err != nil {
		return errorsx.MaybeLog(errors.Wrapf(err, "failed to write certificate to %s", certpath))
	}

	log.Println("writing private key", keypath)
	if err = os.WriteFile(keypath, key, 0600); err != nil {
		return errorsx.MaybeLog(errors.Wrapf(err, "failed to write private key to %s", keypath))
	}
	return nil
}
