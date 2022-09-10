package certificatecache

import (
	"log"
	"os"
	"os/user"
	"path/filepath"

	"github.com/hashicorp/vault/api"
	"github.com/james-lawrence/bw/internal/x/stringsx"
	"github.com/pkg/errors"
)

// VaultDefaultTokenPath returns the path to the default location of the vault token.
func VaultDefaultTokenPath() string {
	var (
		err error
		u   *user.User
	)

	if u, err = user.Current(); err != nil {
		log.Println("failed to lookup the current user, unable to generate default token path", err)
		return ""
	}

	return filepath.Join(u.HomeDir, ".vault-token")
}

// Vault refresh credentials from vault. To use vault the following values need
// to be specified in the configuration file.
// credentialsSource = "vault"
// vaultPKIPath = "path/to/pki/issue"
// servername = "example.com"
type Vault struct {
	CertificateDir   string
	Path             string `yaml:"vaultPKIPath"` // path to the vault PKI to use for credentials.
	CommonName       string `yaml:"servername"`   // common name for certificate, usually a domain name. pulls from the servername of the configuration.
	DefaultTokenFile string // path to the fallback token file.
}

// Refresh the credentials if necessary.
func (t Vault) Refresh() (err error) {
	var (
		client      *api.Client
		credentials *api.Secret
		config      *api.Config
	)

	if config = api.DefaultConfig(); config.Error != nil {
		return errors.WithStack(config.Error)
	}

	if client, err = api.NewClient(config); err != nil {
		return errors.WithStack(err)
	}
	client.SetToken(stringsx.DefaultIfBlank(client.Token(), t.readTokenFile()))

	payload := map[string]interface{}{
		"common_name": t.CommonName,
	}

	if credentials, err = client.Logical().Write(t.Path, payload); err != nil {
		return errors.WithStack(err)
	}

	log.Println("credentials fingerprint", credentials.Data["serial_number"])

	capath := filepath.Join(t.CertificateDir, DefaultTLSCertCA)
	keypath := filepath.Join(t.CertificateDir, DefaultTLSKeyServer)
	certpath := filepath.Join(t.CertificateDir, DefaultTLSCertServer)

	log.Println("writing private key", keypath)
	if err = os.WriteFile(keypath, []byte(credentials.Data["private_key"].(string)), 0600); err != nil {
		return errors.Wrapf(err, "failed to write private key to %s", keypath)
	}

	log.Println("writing certificate", certpath)
	if err = os.WriteFile(certpath, []byte(credentials.Data["certificate"].(string)), 0600); err != nil {
		return errors.Wrapf(err, "failed to write certificate to %s", certpath)
	}

	log.Println("writing authority certificate", capath)
	if err = os.WriteFile(capath, []byte(credentials.Data["issuing_ca"].(string)), 0600); err != nil {
		return errors.Wrapf(err, "failed to write certificate authority to %s", capath)
	}

	return nil
}

func (t Vault) readTokenFile() string {
	var (
		err error
		raw []byte
	)

	if raw, err = os.ReadFile(t.DefaultTokenFile); err != nil {
		log.Println("failed to read vault token from file", t.DefaultTokenFile, err)
		return ""
	}

	return string(raw)
}
