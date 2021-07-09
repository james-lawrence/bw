package daemons

import (
	"crypto/tls"
	"crypto/x509"
	"os"

	"github.com/james-lawrence/bw/agent"
	"github.com/james-lawrence/bw/agent/acme"
	"github.com/james-lawrence/bw/certificatecache"
	"github.com/james-lawrence/bw/internal/x/stringsx"
	"github.com/james-lawrence/bw/internal/x/tlsx"
	"google.golang.org/grpc/credentials"

	"github.com/pkg/errors"
)

// AgentCertificateCache initializes the certificate cache manager.
func AgentCertificateCache(ctx Context) (err error) {
	config := ctx.Config
	client := acme.NewChallenger(ctx.Cluster.Local(), ctx.Cluster, ctx.ACMECache, ctx.Dialer)
	fallback := certificatecache.NewRefreshAgent(config.CredentialsDir, client)

	return certificatecache.FromConfig(
		stringsx.DefaultIfBlank(config.CredentialsDir, config.Credentials.Directory),
		stringsx.DefaultIfBlank(config.CredentialsMode, config.Credentials.Mode),
		ctx.ConfigurationFile,
		fallback,
	)
}

// TLSGenServer generate tls config for the agent.
func TLSGenServer(c agent.Config, options ...tlsx.Option) (creds *tls.Config, err error) {
	var (
		pool *x509.CertPool
	)

	if err = os.MkdirAll(c.CredentialsDir, 0700); err != nil {
		return creds, errors.WithStack(err)
	}

	if pool, err = x509.SystemCertPool(); err != nil {
		return creds, errors.WithStack(err)
	}

	m := certificatecache.NewDirectory(
		c.ServerName,
		c.CredentialsDir,
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

	for _, opt := range options {
		if err = opt(creds); err != nil {
			return creds, err
		}
	}

	return creds, nil
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
func TLSGenClient(c agent.ConfigClient) (creds *tls.Config, err error) {
	var (
		pool *x509.CertPool
	)

	if pool, err = x509.SystemCertPool(); err != nil {
		return creds, errors.WithStack(err)
	}

	creds = &tls.Config{
		ServerName:         c.ServerName,
		RootCAs:            pool,
		NextProtos:         []string{"bw.mux"},
		InsecureSkipVerify: c.Credentials.Insecure,
	}

	return creds, nil
}

// GRPCGenClientNoClientCert ...
func GRPCGenClientNoClientCert(c agent.ConfigClient) (credentials.TransportCredentials, error) {
	var (
		err      error
		tlscreds *tls.Config
	)

	if tlscreds, err = TLSGenClient(c); err != nil {
		return nil, err
	}

	tlscreds.GetClientCertificate = nil

	return credentials.NewTLS(tlscreds), nil
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
