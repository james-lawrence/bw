package certificatecache

import (
	"crypto/x509"
	"encoding/pem"
	"io/ioutil"
	"log"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/james-lawrence/bw"
	"github.com/james-lawrence/bw/x/logx"
	"github.com/pkg/errors"
)

const (
	// DefaultTLSCredentialsRoot default name of the parent directory for the credentials
	DefaultTLSCredentialsRoot = bw.DefaultEnvironmentName
	// DefaultTLSKeyCA default name for the certificate authority key.
	DefaultTLSKeyCA = "tlsca.key"
	// DefaultTLSCertCA default name for the certificate authority certificate.
	DefaultTLSCertCA = "tlsca.cert"
	// DefaultTLSKeyClient ...
	DefaultTLSKeyClient = "tlsclient.key"
	// DefaultTLSCertClient ...
	DefaultTLSCertClient = "tlsclient.cert"
	// DefaultTLSKeyServer ...
	DefaultTLSKeyServer = "tlsserver.key"
	// DefaultTLSCertServer ...
	DefaultTLSCertServer = "tlsserver.cert"
)

const (
	// ModeVault refresh certificates using vault's PKI
	ModeVault = "vault"
)

// FromConfig will automatically refresh credentials in the provided directory
// based on the mode and the configuration file.
func FromConfig(dir, mode, configname string) (err error) {
	switch mode {
	case ModeVault:
		v := Vault{
			DefaultTokenFile: VaultDefaultTokenPath(),
			CertificateDir:   dir,
		}

		if err = bw.ExpandAndDecodeFile(configname, &v); err != nil {
			return err
		}

		if strings.TrimSpace(v.CommonName) == "" {
			return errors.New("server name cannot be blank for vault, please set servername in the configuration")
		}

		if strings.TrimSpace(v.Path) == "" {
			return errors.New("vault PKI path cannot be blank, please set VaultPKIPath in the configuration")
		}

		return RefreshAutomatic(dir, v)
	default:
		log.Println("using nop refresh mode, certificates will need to be refreshed manually")
		certpath := bw.LocateFirstInDir(dir, DefaultTLSCertServer, DefaultTLSCertClient)

		// certificate must exist when using nop refresher.
		if _, err := os.Stat(certpath); os.IsNotExist(err) {
			return err
		}

		return RefreshAutomatic(dir, nopRefresh{})
	}
}

// used to refresh credentials.
type refresher interface {
	Refresh() error
}

type nopRefresh struct{}

func (t nopRefresh) Refresh() error {
	return nil
}

// RefreshAutomatic will automatically refresh credentials in the background.
// error is returned if something goes wrong prior to starting the goroutine.
// once the goroutine is started it will return nil.
func RefreshAutomatic(dir string, r refresher) (err error) {
	const (
		window = 3 * time.Hour
	)

	certpath := bw.LocateFirstInDir(dir, DefaultTLSCertServer, DefaultTLSCertClient)

	if err = RefreshExpired(certpath, time.Now().Add(window), r); err != nil {
		return err
	}

	go func() {
		for t := range time.Tick(time.Hour) {
			logx.MaybeLog(errors.Wrap(RefreshExpired(certpath, t.Add(window), r), "failed to refresh credentials"))
		}
	}()

	return nil
}

// RefreshNow will refresh the credentials immediately
func RefreshNow(dir string, r refresher) (err error) {
	// first ensure directory exists.
	if err = os.MkdirAll(dir, 0700); err != nil {
		return errors.WithStack(err)
	}

	return r.Refresh()
}

// RefreshExpired refreshes certificates if the certificate at the provided path
// has an expiration after the provided time.
func RefreshExpired(certpath string, t time.Time, r refresher) (err error) {
	var (
		expiration time.Time
	)

	// first ensure directory exists.
	if err = os.MkdirAll(filepath.Dir(certpath), 0700); err != nil {
		return errors.WithStack(err)
	}

	// force refresh a new certificate if no certificate exists.
	if _, err = os.Stat(certpath); os.IsNotExist(err) {
		return r.Refresh()
	}

	if expiration, err = expiredCert(certpath); err != nil {
		return err
	}

	if t.Equal(expiration) || t.After(expiration) {
		return r.Refresh()
	}

	return nil
}

// returns the expiration of the certificate at the given path.
func expiredCert(path string) (expiration time.Time, err error) {
	var (
		data []byte
		p    *pem.Block
		cert *x509.Certificate
	)

	if data, err = ioutil.ReadFile(path); err != nil {
		log.Println("failed to read certificate", err)
		return expiration, errors.WithStack(err)
	}

	if p, _ = pem.Decode(data); p == nil {
		log.Println("unable to pem decode certificate")
		return expiration, errors.WithStack(err)
	}

	if cert, err = x509.ParseCertificate(p.Bytes); err != nil {
		log.Println("failed parse certificate", err)
		return expiration, errors.WithStack(err)
	}

	log.Println("cert expires at", cert.NotAfter)
	log.Println("cert not valid before", cert.NotBefore)

	return cert.NotAfter, nil
}
