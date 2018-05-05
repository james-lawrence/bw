package certificatecache

import (
	"os"

	"github.com/james-lawrence/bw"
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

// used to refresh credentials.
type refresher interface {
	Refresh() error
}

// RefreshAutomatic will automatically refresh credentials in the background.
// error is returned if something goes wrong prior to starting the goroutine.
// once the goroutine is started it will return nil.
func RefreshAutomatic(dir string, r refresher) (err error) {
	// first ensure directory exists.
	if err = os.MkdirAll(dir, 0700); err != nil {
		return errors.WithStack(err)
	}

	// TODO: check if credentials are expired or nearing their expiration.
	// TODO: start go routine to monitor the credentials to ensure they do not expire.
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
