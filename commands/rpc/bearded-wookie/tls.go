package main

import (
	"crypto/sha256"
	"crypto/tls"
	"crypto/x509"
	"io/ioutil"
	"log"
	"os"
	"os/user"
	"path/filepath"

	"bitbucket.org/jatone/bearded-wookie/x/stringsx"

	"github.com/pkg/errors"
	"google.golang.org/grpc/credentials"
)

func configDirectory(env string) string {
	return filepath.Join(configDirDefault, env)
}

func defaultUserCredentialsDirectory(u *user.User, credentials string) string {
	return filepath.Join(
		stringsx.DefaultIfBlank(os.Getenv(envConfigurationDirectory), filepath.Join(u.HomeDir, ".config")),
		credentialsDirDefault,
		stringsx.DefaultIfBlank(credentials, credentialsDefault),
	)
}

func newDefaultClientTLS(rootdir string) TLSConfig {
	return TLSConfig{
		Key:  filepath.Join(rootdir, tlsclientKeyDefault),
		Cert: filepath.Join(rootdir, tlsclientCertDefault),
		CA:   filepath.Join(rootdir, tlscaCertDefault),
	}
}

func defaultAgentConfigFile(u *user.User) string {
	userRoot := filepath.Join(
		stringsx.DefaultIfBlank(os.Getenv(envConfigurationDirectory), filepath.Join(u.HomeDir, ".config")),
		credentialsDirDefault,
	)
	systemRoot := filepath.Join("/etc", credentialsDirDefault)
	return locateFile("agent.config", userRoot, systemRoot)
}

func newDefaultSystemServerTLS(u *user.User, credentials string) TLSConfig {
	userRoot := defaultUserCredentialsDirectory(u, credentials)
	systemRoot := filepath.Join("/etc", credentialsDirDefault)
	return TLSConfig{
		Key:  locateFile(tlsserverKeyDefault, userRoot, systemRoot),
		Cert: locateFile(tlsserverCertDefault, userRoot, systemRoot),
		CA:   locateFile(tlscaCertDefault, userRoot, systemRoot),
	}
}

func locateFile(name string, searchDirs ...string) (result string) {
	for _, dir := range searchDirs {
		result = filepath.Join(dir, name)
		log.Println("checking", result)
		if _, err := os.Stat(result); err == nil {
			log.Println("found", result)
			break
		}
	}
	return result
}

func newDefaultServerTLS(u *user.User) TLSConfig {
	rootdir := stringsx.DefaultIfBlank(os.Getenv(envConfigurationDirectory), filepath.Join(u.HomeDir, ".config"))
	rootdir = filepath.Join(rootdir, credentialsDirDefault, credentialsDefault)
	return TLSConfig{
		Key:  filepath.Join(rootdir, tlsserverKeyDefault),
		Cert: filepath.Join(rootdir, tlsserverCertDefault),
		CA:   filepath.Join(rootdir, tlscaCertDefault),
	}
}

// TLSConfig ...
type TLSConfig struct {
	Key        string
	Cert       string
	CA         string
	ServerName string
}

func (t TLSConfig) fingerprint() (raw []byte, err error) {
	compute := sha256.New()

	if raw, err = ioutil.ReadFile(t.Key); err != nil {
		return raw, errors.WithStack(err)
	}

	if _, err = compute.Write(raw); err != nil {
		return raw, errors.WithStack(err)
	}

	return compute.Sum(nil), nil
}

func (t TLSConfig) buildServer() (creds credentials.TransportCredentials, err error) {
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

func (t TLSConfig) buildClient() (creds credentials.TransportCredentials, err error) {
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
