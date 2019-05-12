package certificatecache

import (
	"crypto"
	"crypto/md5"
	"crypto/rsa"
	"crypto/x509"
	"encoding/hex"
	"encoding/json"
	"encoding/pem"
	"io/ioutil"
	"log"
	"path/filepath"
	"strconv"

	"github.com/go-acme/lego/certcrypto"
	"github.com/go-acme/lego/certificate"
	"github.com/go-acme/lego/challenge/tlsalpn01"
	"github.com/go-acme/lego/lego"
	"github.com/go-acme/lego/registration"
	"github.com/pkg/errors"
)

// export LEGO_CA_CERTIFICATES="${HOME}/.golang/lib/src/github.com/letsencrypt/pebble/test/certs/pebble.minica.pem"
// cd ${HOME}/.golang/lib/src/github.com/letsencrypt/pebble; pebble -config ./test/config/pebble-config.json
// func main() {
// 	_, err := commandutils.LoadConfiguration(bw.DefaultEnvironmentName)
// 	if err != nil {
// 		log.Fatalln(err)
// 	}
// 	<-make(chan struct{})
// }

type acmeUser struct {
	Email        string
	Registration *registration.Resource
	key          crypto.PrivateKey
}

func (u acmeUser) GetEmail() string {
	return u.Email
}

func (u acmeUser) GetRegistration() *registration.Resource {
	return u.Registration
}

func (u acmeUser) GetPrivateKey() crypto.PrivateKey {
	return u.key
}

// ACMEConfig configuration for ACME credentials
type ACMEConfig struct {
	CAURL      string `yaml:"caurl"`
	Email      string `yaml:"email"`
	PrivateKey string `yaml:"key"` // PEM encoded private key.,
	Port       int    `yaml:"port"`
	Network    string `yaml:"network"`
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
		client *lego.Client
		u      = acmeUser{
			Email: t.Config.Email,
		}
	)

	if u.key, err = t.generatePrivateKey(); err != nil {
		return errors.Wrap(err, "failed to generate or load private key")
	}

	config := lego.NewConfig(&u)

	config.CADirURL = t.Config.CAURL
	config.Certificate.KeyType = certcrypto.RSA8192

	if client, err = lego.NewClient(config); err != nil {
		return errors.Wrap(err, "failed to build acme client")
	}

	log.Println("client created")

	if u.Registration, err = t.loadRegistration(client); err != nil {
		return errors.Wrap(err, "failed to load ACME registration")
	}

	log.Println("loaded registration")

	if err = client.Challenge.SetTLSALPN01Provider(tlsalpn01.NewProviderServer(t.Config.Network, strconv.Itoa(t.Config.Port))); err != nil {
		return errors.Wrap(err, "failed to setup tlsalpn01 provider")
	}

	request := certificate.ObtainRequest{
		Domains: []string{"localhost"},
		Bundle:  true,
	}

	log.Println("obtaining certificate")
	certificates, err := client.Certificate.Obtain(request)
	if err != nil {
		return errors.Wrap(err, "failed to obtain certificates")
	}

	log.Println("obtained certificate")

	capath := filepath.Join(t.CertificateDir, DefaultTLSCertCA)
	keypath := filepath.Join(t.CertificateDir, DefaultTLSKeyServer)
	certpath := filepath.Join(t.CertificateDir, DefaultTLSCertServer)

	log.Println("writing private key", keypath)
	if err = ioutil.WriteFile(keypath, certificates.PrivateKey, 0600); err != nil {
		return errors.Wrapf(err, "failed to write private key to %s", keypath)
	}

	log.Println("writing certificate", certpath)
	if err = ioutil.WriteFile(certpath, certificates.Certificate, 0600); err != nil {
		return errors.Wrapf(err, "failed to write certificate to %s", certpath)
	}

	log.Println("writing authority certificate", capath)
	if err = ioutil.WriteFile(capath, certificates.IssuerCertificate, 0600); err != nil {
		return errors.Wrapf(err, "failed to write certificate authority to %s", capath)
	}

	// TODO: do we need to do anything w/ these
	log.Println(certificates.Domain, certificates.CertURL, certificates.CertStableURL)

	return nil
}

func (t ACME) generatePrivateKey() (priv *rsa.PrivateKey, err error) {
	b, _ := pem.Decode([]byte(t.Config.PrivateKey))
	if priv, err = x509.ParsePKCS1PrivateKey(b.Bytes); err != nil {
		return nil, errors.WithStack(err)
	}

	return priv, nil
}

func (t ACME) loadRegistration(client *lego.Client) (reg *registration.Resource, err error) {
	var (
		encoded []byte
	)

	regp := filepath.Join(t.CertificateDir, "acme.registration.json")

	if reg, err = client.Registration.Register(registration.RegisterOptions{TermsOfServiceAgreed: true}); err != nil {
		return nil, err
	}

	if encoded, err = json.Marshal(reg); err != nil {
		return nil, err
	}

	if err = ioutil.WriteFile(regp, encoded, 0600); err != nil {
		return nil, err
	}

	return reg, nil
}
