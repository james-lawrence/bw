package daemons

import (
	"crypto/tls"
	"crypto/x509"
	"io/ioutil"

	"github.com/james-lawrence/bw/agent"
	"github.com/james-lawrence/bw/certificatecache"
	"google.golang.org/grpc/credentials"

	"github.com/pkg/errors"
)

// TLSGenServer generate tls config for the agent.
func TLSGenServer(c agent.Config) (creds *tls.Config, err error) {
	var (
		ca   []byte
		pool *x509.CertPool
	)

	m := certificatecache.NewDirectory(c.ServerName, c.CredentialsDir)

	if pool, err = x509.SystemCertPool(); err != nil {
		return creds, errors.WithStack(err)
	}

	if ca, err = ioutil.ReadFile(c.CA); err != nil {
		return creds, errors.WithStack(err)
	}

	if ok := pool.AppendCertsFromPEM(ca); !ok {
		return creds, errors.New("failed to append client ca")
	}

	return &tls.Config{
		ServerName:           c.ServerName,
		ClientAuth:           tls.RequireAndVerifyClientCert,
		GetCertificate:       m.GetCertificate,
		GetClientCertificate: m.GetClientCertificate,
		ClientCAs:            pool,
		RootCAs:              pool,
	}, nil
}

// GRPCGenServer generate grpc tls transport credentials for the server.
func GRPCGenServer(c agent.Config) (credentials.TransportCredentials, error) {
	var (
		err      error
		tlscreds *tls.Config
	)

	if tlscreds, err = TLSGenServer(c); err != nil {
		return nil, err
	}

	return credentials.NewTLS(tlscreds), nil
}

// TLSGenClient generate tls config for a client.
func TLSGenClient(c agent.ConfigClient) (creds *tls.Config, err error) {
	var (
		ca   []byte
		pool *x509.CertPool
	)

	m := certificatecache.NewDirectory(c.ServerName, c.CredentialsDir)

	if pool, err = x509.SystemCertPool(); err != nil {
		return creds, errors.WithStack(err)
	}

	if ca, err = ioutil.ReadFile(c.CA); err != nil {
		return creds, errors.WithStack(err)
	}

	if ok := pool.AppendCertsFromPEM(ca); !ok {
		return creds, errors.New("failed to append client certs")
	}

	creds = &tls.Config{
		ServerName:           c.ServerName,
		RootCAs:              pool,
		GetCertificate:       m.GetCertificate,
		GetClientCertificate: m.GetClientCertificate,
	}

	return creds, nil
}

// GRPCGenClient generate grpc tls transport credentials for a client.
func GRPCGenClient(c agent.ConfigClient) (credentials.TransportCredentials, error) {
	var (
		err      error
		tlscreds *tls.Config
	)

	if tlscreds, err = TLSGenClient(c); err != nil {
		return nil, err
	}

	return credentials.NewTLS(tlscreds), nil
}
