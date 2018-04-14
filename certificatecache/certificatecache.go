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
