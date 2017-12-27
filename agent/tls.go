package agent

import (
	"crypto/tls"
	"crypto/x509"
	"io/ioutil"
	"log"
	"path/filepath"

	"github.com/james-lawrence/bw"
	"github.com/james-lawrence/bw/x/systemx"

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

	log.Println("loading client cert", t.Cert)
	log.Println("loading client key", t.Key)
	log.Println("loading authority cert", t.CA)
	log.Println("using server name", t.ServerName)

	creds = &tls.Config{
		ServerName:   t.ServerName,
		ClientAuth:   tls.RequireAndVerifyClientCert,
		Certificates: []tls.Certificate{cert},
		ClientCAs:    pool,
		RootCAs:      pool,
	}

	return creds, nil
}

// BuildClient ...
func (t ConfigClient) BuildClient() (creds *tls.Config, err error) {
	var (
		cert tls.Certificate
		ca   []byte
	)

	log.Println("loading client cert", t.Cert)
	log.Println("loading client key", t.Key)
	log.Println("loading authority cert", t.CA)
	log.Println("using server name", t.ServerName)
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
