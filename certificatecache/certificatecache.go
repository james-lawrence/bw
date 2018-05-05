package certificatecache

import "github.com/james-lawrence/bw"

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

// // AutomaticRefresh will automatically refresh credentials in the background.
// // error is returned if something goes wrong prior to starting the goroutine.
// // once the goroutine is started it will return nil.
// func AutomaticRefresh(dir string, r refresher) error {
// 	return nil
// }
