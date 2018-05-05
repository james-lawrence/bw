package agent

import (
	"crypto/tls"
	"crypto/x509"
	"io/ioutil"
	"path/filepath"

	"github.com/james-lawrence/bw"
	"github.com/james-lawrence/bw/certificatecache"
	"github.com/james-lawrence/bw/x/systemx"
	"google.golang.org/grpc/credentials"

	"github.com/pkg/errors"
)

// ConfigClientTLS ...
func ConfigClientTLS(credentials string) ConfigClientOption {
	return func(c *ConfigClient) {
		c.CredentialsDir = bw.DefaultLocation(credentials, "")
		c.CA = bw.DefaultLocation(filepath.Join(credentials, certificatecache.DefaultTLSCertCA), "")
		c.ServerName = systemx.HostnameOrLocalhost()
	}
}

// NewTLSAgent ...
func newTLSAgent(credentials, override string) ConfigOption {
	return func(c *Config) {
		c.CA = bw.DefaultLocation(filepath.Join(credentials, certificatecache.DefaultTLSCertCA), override)
		c.CredentialsDir = bw.DefaultLocation(credentials, override)
		c.ServerName = systemx.HostnameOrLocalhost()
	}
}

// BuildServer ...
func (t Config) BuildServer() (creds *tls.Config, err error) {
	var (
		ca []byte
	)

	m := certificatecache.NewDirectory(t.ServerName, t.CredentialsDir)
	pool := x509.NewCertPool()

	if ca, err = ioutil.ReadFile(t.CA); err != nil {
		return creds, errors.WithStack(err)
	}

	if ok := pool.AppendCertsFromPEM(ca); !ok {
		return creds, errors.New("failed to append client ca")
	}

	return &tls.Config{
		ServerName:           t.ServerName,
		ClientAuth:           tls.RequireAndVerifyClientCert,
		GetCertificate:       m.GetCertificate,
		GetClientCertificate: m.GetClientCertificate,
		ClientCAs:            pool,
		RootCAs:              pool,
	}, nil
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
		ca []byte
	)

	m := certificatecache.NewDirectory(t.ServerName, t.CredentialsDir)
	pool := x509.NewCertPool()
	if ca, err = ioutil.ReadFile(t.CA); err != nil {
		return nil, errors.WithStack(err)
	}

	if ok := pool.AppendCertsFromPEM(ca); !ok {
		return nil, errors.New("failed to append client certs")
	}

	creds = &tls.Config{
		ServerName:           t.ServerName,
		RootCAs:              pool,
		GetCertificate:       m.GetCertificate,
		GetClientCertificate: m.GetClientCertificate,
	}

	return creds, nil
}
