package certificatecache

import (
	"crypto/tls"
	"crypto/x509"
	"os"

	"github.com/james-lawrence/bw/agent"
	"github.com/james-lawrence/bw/internal/systemx"
	"github.com/james-lawrence/bw/internal/tlsx"
	"google.golang.org/grpc/credentials"

	"github.com/pkg/errors"
)

// TLSGenServer generate tls config for the agent.
func TLSGenServer(c agent.Config, options ...tlsx.Option) (creds *tls.Config, err error) {
	var (
		pool *x509.CertPool
	)

	if err = os.MkdirAll(c.Credentials.Directory, 0700); err != nil {
		return creds, errors.WithStack(err)
	}

	if pool, err = x509.SystemCertPool(); err != nil {
		return creds, errors.WithStack(err)
	}

	m := NewDirectory(
		c.ServerName,
		c.Credentials.Directory,
		c.CA,
		pool,
	)

	creds = &tls.Config{
		ServerName:           c.ServerName,
		ClientAuth:           tls.RequireAndVerifyClientCert,
		GetCertificate:       m.GetCertificate,
		GetClientCertificate: m.GetClientCertificate,
		ClientCAs:            pool,
		RootCAs:              pool,
		NextProtos:           []string{"bw.mux"},
	}

	return tlsx.Clone(creds, options...)
}

// GRPCGenServer generate grpc tls transport credentials for the server.
func GRPCGenServer(c agent.Config, options ...tlsx.Option) (credentials.TransportCredentials, error) {
	var (
		err      error
		tlscreds *tls.Config
	)

	if tlscreds, err = TLSGenServer(c, options...); err != nil {
		return nil, err
	}

	return credentials.NewTLS(tlscreds), nil
}

// TLSGenClient generate tls config for a client.
func TLSGenClient(c agent.ConfigClient, options ...tlsx.Option) (creds *tls.Config, err error) {
	var (
		pool *x509.CertPool
	)

	if pool, err = x509.SystemCertPool(); err != nil {
		return creds, errors.WithStack(err)
	}

	if systemx.FileExists(c.CA) {
		if err = LoadCert(pool, c.CA); err != nil {
			return creds, errors.WithStack(err)
		}
	}

	creds = &tls.Config{
		ServerName:         c.ServerName,
		RootCAs:            pool,
		NextProtos:         []string{"bw.mux"},
		InsecureSkipVerify: c.Credentials.Insecure,
	}

	return tlsx.Clone(creds, options...)
}

// GRPCGenClient generate grpc tls transport credentials for a client.
func GRPCGenClient(c agent.ConfigClient, options ...tlsx.Option) (credentials.TransportCredentials, error) {
	var (
		err      error
		tlscreds *tls.Config
	)

	if tlscreds, err = TLSGenClient(c, options...); err != nil {
		return nil, err
	}

	return credentials.NewTLS(tlscreds), nil
}
