package certificatecache

import (
	"crypto"
	"log"

	"github.com/davecgh/go-spew/spew"
	"github.com/pkg/errors"
	"github.com/xenolf/lego/acme"
)

type acmeUser struct {
	Email        string
	Registration *acme.RegistrationResource
	key          crypto.PrivateKey
}

func (u acmeUser) GetEmail() string {
	return u.Email
}
func (u acmeUser) GetRegistration() *acme.RegistrationResource {
	return u.Registration
}
func (u acmeUser) GetPrivateKey() crypto.PrivateKey {
	return u.key
}

// ACME protocol certificate creation.
type ACME struct {
	CertificateDir string
	CommonName     string `yaml:"servername"` // common name for certificate, usually a domain name. pulls from the servername of the configuration.
	PrivateKey     string
}

// Refresh the credentials if necessary.
func (t ACME) Refresh() (err error) {
	var (
		u = acmeUser{}
	)

	client, err := acme.NewClient("https://acme-staging-v02.api.letsencrypt.org/directory", &u, acme.RSA2048)
	if err != nil {
		return errors.Wrap(err, "failed to build acme client")
	}

	// New users will need to register
	if u.Registration, err = client.Register(true); err != nil {
		return errors.Wrap(err, "failed to register acme user")
	}

	log.Println("registration", spew.Sdump(u.Registration))

	return nil
}
