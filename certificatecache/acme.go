package certificatecache

import (
	"crypto"
	"crypto/md5"
	"crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/asn1"
	"encoding/hex"
	"encoding/json"
	"io/ioutil"
	"log"
	"path/filepath"
	"strconv"

	"github.com/go-acme/lego/certcrypto"
	"github.com/go-acme/lego/challenge/tlsalpn01"
	"github.com/go-acme/lego/lego"
	"github.com/go-acme/lego/registration"
	"github.com/pkg/errors"

	"github.com/james-lawrence/bw"
	"github.com/james-lawrence/bw/internal/x/sshx"
)

// export LEGO_CA_CERTIFICATES="${HOME}/go/src/github.com/letsencrypt/pebble/test/certs/pebble.minica.pem"
// cd ${HOME}/go/src/github.com/letsencrypt/pebble; pebble -config ./test/config/pebble-config.json

func defaultConfig() ACMEConfig {
	return ACMEConfig{
		CAURL: lego.LEDirectoryProduction,
		Port:  bw.DefaultACMEPort,
		reg:   &registration.Resource{},
	}
}

// ACMEConfig configuration for ACME credentials
type ACMEConfig struct {
	CAURL              string   `yaml:"caurl"`
	Email              string   `yaml:"email"`
	PrivateKey         string   `yaml:"key"` // PEM encoded private key.
	Port               int      `yaml:"port"`
	Network            string   `yaml:"network"`
	Country            []string `yaml:"country"`  // Country Codes for the CSR
	Province           []string `yaml:"province"` // Provinces for the CSR
	Locality           []string `yaml:"locality"`
	Organization       []string `yaml:"organization"`
	OrganizationalUnit []string `yaml:"organizationalUnit"`
	DNSNames           []string `yaml:"dns"` // alternative dns names
	reg                *registration.Resource
}

// GetPrivateKey implement config for acme lego.
func (t ACMEConfig) GetPrivateKey() (priv crypto.PrivateKey) {
	var (
		err error
	)

	if priv, err = sshx.DecodeRSA([]byte(t.PrivateKey)); err != nil {
		log.Println("failed to decode private key", err)
		return nil
	}

	return priv
}

// GetRegistration ...
func (t ACMEConfig) GetRegistration() *registration.Resource {
	return t.reg
}

// GetEmail ...
func (t ACMEConfig) GetEmail() string {
	return t.Email
}

// ACME provides the ability to generate certificates using the acme protocol.
type ACME struct {
	CertificateDir string
	CommonName     string     `yaml:"servername"` // common name for certificate, usually a domain name. pulls from the servername of the configuration.
	Config         ACMEConfig `yaml:"acme"`
}

func (t ACME) sanitize() ACME {
	digest := md5.Sum([]byte(t.Config.PrivateKey))
	t.Config.PrivateKey = "fingerprint:" + hex.EncodeToString(digest[:])
	return t
}

// Refresh the credentials if necessary.
func (t ACME) Refresh() (err error) {
	var (
		encoded []byte
		client  *lego.Client
		priv    *rsa.PrivateKey
	)

	if len(t.Config.PrivateKey) == 0 {
		if encoded, err = sshx.CachedAuto(filepath.Join(t.CertificateDir, defaultACMEKey)); err != nil {
			return err
		}

		t.Config.PrivateKey = string(encoded)
	}

	config := lego.NewConfig(t.Config)

	config.CADirURL = t.Config.CAURL
	config.Certificate.KeyType = certcrypto.RSA8192

	if client, err = lego.NewClient(config); err != nil {
		return errors.Wrap(err, "failed to build acme client")
	}

	log.Println("client created")

	if *t.Config.reg, err = t.loadRegistration(client); err != nil {
		return errors.Wrap(err, "failed to load ACME registration")
	}

	log.Println("loaded registration")

	if err = client.Challenge.SetTLSALPN01Provider(tlsalpn01.NewProviderServer(t.Config.Network, strconv.Itoa(t.Config.Port))); err != nil {
		return errors.Wrap(err, "failed to setup tlsalpn01 provider")
	}

	if priv, err = sshx.MaybeDecodeRSA(sshx.CachedAuto(filepath.Join(t.CertificateDir, DefaultTLSKeyServer))); err != nil {
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

	cert, err := x509.CreateCertificateRequest(rand.Reader, template, priv)
	if err != nil {
		return errors.Wrap(err, "failed to create CSR")
	}

	if template, err = x509.ParseCertificateRequest(cert); err != nil {
		return errors.Wrap(err, "failed to decode CSR")
	}

	log.Println("obtaining certificate")
	certificates, err := client.Certificate.ObtainForCSR(*template, true)
	if err != nil {
		return errors.Wrap(err, "failed to obtain certificates")
	}

	log.Println("obtained certificate")

	capath := filepath.Join(t.CertificateDir, DefaultTLSCertCA)
	certpath := filepath.Join(t.CertificateDir, DefaultTLSCertServer)

	log.Println("writing authority certificate", capath)
	if err = ioutil.WriteFile(capath, certificates.IssuerCertificate, 0600); err != nil {
		return errors.Wrapf(err, "failed to write certificate authority to %s", capath)
	}

	log.Println("writing certificate", certpath)
	if err = ioutil.WriteFile(certpath, certificates.Certificate, 0600); err != nil {
		return errors.Wrapf(err, "failed to write certificate to %s", certpath)
	}

	// TODO: do we need to do anything w/ these
	log.Println(certificates.Domain, certificates.CertURL, certificates.CertStableURL)

	return nil
}

func (t ACME) loadRegistration(client *lego.Client) (zreg registration.Resource, err error) {
	var (
		encoded []byte
		reg     *registration.Resource
	)

	regp := filepath.Join(t.CertificateDir, "acme.registration.json")

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
