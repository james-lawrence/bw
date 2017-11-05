package main

import (
	"crypto/rsa"
	"crypto/x509"
	"crypto/x509/pkix"
	"os"
	"path/filepath"
	"time"

	"bitbucket.org/jatone/bearded-wookie"
	"bitbucket.org/jatone/bearded-wookie/agent"
	"bitbucket.org/jatone/bearded-wookie/x/tlsx"
	"github.com/alecthomas/kingpin"
	"github.com/pkg/errors"
)

// bearded-wookie init talla.io *.talla.io 127.0.0.1 127.0.0.2
// bearded-wookie init
// bearded-wookie init talla.io wambli.talla.io
// bearded-wookie init talla.io wambli.talla.io

type initCmd struct {
	global      *global
	credentials string
	duration    time.Duration
	hosts       []string
	bits        int
	common      string
}

func (t *initCmd) configure(parent *kingpin.CmdClause) {
	parent.Flag("duration", "how long the certificate should last").Default("8760h").DurationVar(&t.duration)
	parent.Flag("rsa-bits", "size of RSA key to generate.").Default("4096").IntVar(&t.bits)
	parent.Flag("credentials", "name of the credentials to generate").Default("default").StringVar(&t.credentials)
	parent.Arg("common-name", "common name of the authority").StringVar(&t.common)
	parent.Arg("hosts", "hosts the certificate should match").StringsVar(&t.hosts)
	parent.Action(t.generate)
}

func (t *initCmd) generate(ctx *kingpin.ParseContext) (err error) {
	var (
		capriv    *rsa.PrivateKey
		authority x509.Certificate
		server    x509.Certificate
		client    x509.Certificate
		rootdir   = bw.DefaultDirLocation(t.credentials)
	)
	defer t.global.shutdown()

	caoptions := []tlsx.X509Option{
		tlsx.X509OptionCA(),
		tlsx.X509OptionSubject(pkix.Name{
			CommonName: t.common,
		}),
	}

	servoptions := []tlsx.X509Option{
		tlsx.X509OptionUsage(x509.KeyUsageKeyEncipherment | x509.KeyUsageDigitalSignature | x509.KeyUsageKeyAgreement),
		tlsx.X509OptionUsageExt(x509.ExtKeyUsageAny),
		tlsx.X509OptionHosts(t.hosts...),
		tlsx.X509OptionSubject(pkix.Name{
			CommonName: "server",
		}),
	}

	clientoptions := []tlsx.X509Option{
		tlsx.X509OptionUsage(x509.KeyUsageKeyEncipherment | x509.KeyUsageDigitalSignature | x509.KeyUsageKeyAgreement),
		tlsx.X509OptionUsageExt(x509.ExtKeyUsageClientAuth),
		tlsx.X509OptionHosts(t.hosts...),
		tlsx.X509OptionSubject(pkix.Name{
			CommonName: "client",
		}),
	}

	if authority, err = tlsx.X509Template(t.duration, caoptions...); err != nil {
		return err
	}

	if server, err = tlsx.X509Template(t.duration, servoptions...); err != nil {
		return err
	}

	if client, err = tlsx.X509Template(t.duration, clientoptions...); err != nil {
		return err
	}

	if err = os.MkdirAll(rootdir, 0755); err != nil {
		return errors.WithStack(err)
	}

	write := func(key, cert string) func(*rsa.PrivateKey, []byte, error) (*rsa.PrivateKey, error) {
		var (
			keydst  *os.File
			certdst *os.File
		)
		if keydst, err = os.OpenFile(filepath.Join(rootdir, key), os.O_CREATE|os.O_RDWR|os.O_TRUNC, 0600); err != nil {
			return func(*rsa.PrivateKey, []byte, error) (*rsa.PrivateKey, error) {
				return nil, err
			}
		}

		if certdst, err = os.OpenFile(filepath.Join(rootdir, cert), os.O_CREATE|os.O_RDWR|os.O_TRUNC, 0600); err != nil {
			return func(*rsa.PrivateKey, []byte, error) (*rsa.PrivateKey, error) {
				return nil, err
			}
		}

		return func(priv *rsa.PrivateKey, derBytes []byte, err error) (*rsa.PrivateKey, error) {
			defer keydst.Close()
			defer certdst.Close()

			return priv, tlsx.WriteTLS(priv, derBytes, err)(keydst, certdst, nil)
		}
	}

	if capriv, err = write(agent.DefaultTLSKeyCA, agent.DefaultTLSCertCA)(tlsx.SelfSignedRSAGen(t.bits, authority)); err != nil {
		return err
	}

	if _, err = write(agent.DefaultTLSKeyServer, agent.DefaultTLSCertServer)(tlsx.SignedRSAGen(t.bits, server, authority, capriv)); err != nil {
		return err
	}

	if _, err = write(agent.DefaultTLSKeyClient, agent.DefaultTLSCertClient)(tlsx.SignedRSAGen(t.bits, client, authority, capriv)); err != nil {
		return err
	}

	return err
}
