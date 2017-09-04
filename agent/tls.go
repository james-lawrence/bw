package agent

import (
	"crypto/sha256"
	"crypto/tls"
	"crypto/x509"
	"io/ioutil"
	"log"
	"path/filepath"

	"bitbucket.org/jatone/bearded-wookie"
	"bitbucket.org/jatone/bearded-wookie/x/systemx"

	"github.com/pkg/errors"
	"google.golang.org/grpc/credentials"
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

// NewTLSClient ...
func NewTLSClient(credentials string) TLSConfig {
	return TLSConfig{
		Key:        bw.DefaultLocation(filepath.Join(credentials, DefaultTLSKeyClient), ""),
		Cert:       bw.DefaultLocation(filepath.Join(credentials, DefaultTLSCertClient), ""),
		CA:         bw.DefaultLocation(filepath.Join(credentials, DefaultTLSCertCA), ""),
		ServerName: systemx.HostnameOrLocalhost(),
	}
}

// NewTLSAgent ...
func NewTLSAgent(credentials, override string) TLSConfig {
	return TLSConfig{
		Key:        bw.DefaultLocation(filepath.Join(credentials, DefaultTLSKeyServer), override),
		Cert:       bw.DefaultLocation(filepath.Join(credentials, DefaultTLSCertServer), override),
		CA:         bw.DefaultLocation(filepath.Join(credentials, DefaultTLSCertCA), override),
		ServerName: systemx.HostnameOrLocalhost(),
	}
}

// TLSConfig ...
type TLSConfig struct {
	Key        string
	Cert       string
	CA         string
	ServerName string
}

// Hash - returns the hash of the TLS key.
func (t TLSConfig) Hash() (raw []byte, err error) {
	compute := sha256.New()

	if raw, err = ioutil.ReadFile(t.Key); err != nil {
		return raw, errors.WithStack(err)
	}

	if _, err = compute.Write(raw); err != nil {
		return raw, errors.WithStack(err)
	}

	return compute.Sum(nil), nil
}

// BuildServer ...
func (t TLSConfig) BuildServer() (creds credentials.TransportCredentials, err error) {
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

	creds = credentials.NewTLS(
		&tls.Config{
			ClientAuth:   tls.RequireAndVerifyClientCert,
			Certificates: []tls.Certificate{cert},
			ClientCAs:    pool,
		},
	)

	return creds, nil
}

// BuildClient ...
func (t TLSConfig) BuildClient() (creds credentials.TransportCredentials, err error) {
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

	creds = credentials.NewTLS(
		&tls.Config{
			ServerName:   t.ServerName,
			Certificates: []tls.Certificate{cert},
			RootCAs:      pool,
		},
	)

	return creds, nil
}
