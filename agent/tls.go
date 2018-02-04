package agent

import (
	"crypto/tls"
	"crypto/x509"
	"io/ioutil"
	"path/filepath"

	"github.com/james-lawrence/bw"
	"github.com/james-lawrence/bw/x/systemx"
	"google.golang.org/grpc/credentials"

	"github.com/pkg/errors"
)

const (
	// DefaultTLSCredentialsRoot default name of the parent directory for the credentials
	DefaultTLSCredentialsRoot = "default"
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

// ConfigClientTLS ...
func ConfigClientTLS(credentials string) ConfigClientOption {
	return func(c *ConfigClient) {
		c.Key = bw.DefaultLocation(filepath.Join(credentials, DefaultTLSKeyClient), "")
		c.Cert = bw.DefaultLocation(filepath.Join(credentials, DefaultTLSCertClient), "")
		c.CA = bw.DefaultLocation(filepath.Join(credentials, DefaultTLSCertCA), "")
		c.ServerName = systemx.HostnameOrLocalhost()
	}
}

// NewTLSAgent ...
func newTLSAgent(credentials, override string) ConfigOption {
	return func(c *Config) {
		c.Key = bw.DefaultLocation(filepath.Join(credentials, DefaultTLSKeyServer), override)
		c.Cert = bw.DefaultLocation(filepath.Join(credentials, DefaultTLSCertServer), override)
		c.CA = bw.DefaultLocation(filepath.Join(credentials, DefaultTLSCertCA), override)
		c.ServerName = systemx.HostnameOrLocalhost()
	}
}

// BuildServer ...
func (t Config) BuildServer() (creds *tls.Config, err error) {
	var (
		cert tls.Certificate
		ca   []byte
	)

	if cert, err = tls.LoadX509KeyPair(t.Cert, t.Key); err != nil {
		return creds, errors.WithStack(err)
	}

	pool := x509.NewCertPool()
	if ca, err = ioutil.ReadFile(t.CA); err != nil {
		return creds, errors.WithStack(err)
	}

	if ok := pool.AppendCertsFromPEM(ca); !ok {
		return creds, errors.New("failed to append client certs")
	}

	creds = &tls.Config{
		ServerName:   t.ServerName,
		ClientAuth:   tls.RequireAndVerifyClientCert,
		Certificates: []tls.Certificate{cert},
		ClientCAs:    pool,
		RootCAs:      pool,
	}

	return creds, nil
}

// GRPCCredentials creates grpc transport credentials from the TLS configuration.
func (t Config) GRPCCredentials() (credentials.TransportCredentials, error) {
	var (
		err      error
		tlscreds *tls.Config
	)

	if tlscreds, err = t.BuildServer(); err != nil {
		return nil, err
	}

	return credentials.NewTLS(tlscreds), nil
}

// BuildClient ...
func (t ConfigClient) BuildClient() (creds *tls.Config, err error) {
	var (
		cert tls.Certificate
		ca   []byte
	)

	if cert, err = tls.LoadX509KeyPair(t.Cert, t.Key); err != nil {
		return nil, errors.WithStack(err)
	}

	pool := x509.NewCertPool()
	if ca, err = ioutil.ReadFile(t.CA); err != nil {
		return nil, errors.WithStack(err)
	}

	if ok := pool.AppendCertsFromPEM(ca); !ok {
		return nil, errors.New("failed to append client certs")
	}

	creds = &tls.Config{
		ServerName:   t.ServerName,
		Certificates: []tls.Certificate{cert},
		RootCAs:      pool,
	}

	return creds, nil
}
