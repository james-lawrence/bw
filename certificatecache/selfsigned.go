package certificatecache

import (
	"crypto/rsa"
	"crypto/x509"
	"crypto/x509/pkix"
	"log"
	"path/filepath"
	"time"

	"github.com/james-lawrence/bw"
	"github.com/james-lawrence/bw/internal/envx"
	"github.com/james-lawrence/bw/internal/rsax"
	"github.com/james-lawrence/bw/internal/tlsx"
)

// minimumExpiration is used to force a certificate refresh of self signed certificates.
func minimumExpiration() time.Duration {
	return 24 * time.Hour
}

// generates a self signed certificate iff the current certificate is missing or
// expired. this is used to allow the cluster to bootstrap correctly.
type selfsigned struct {
	seed           []byte
	domain         string
	credentialsDir string
}

func (t selfsigned) Refresh() (err error) {
	var (
		priv     *rsa.PrivateKey
		cert     []byte
		template x509.Certificate
	)

	log.Println("refreshing self signed certificate", t.credentialsDir)
	subject := tlsx.X509OptionSubject(pkix.Name{
		CommonName: t.domain,
	})

	if priv, err = rsax.MaybeDecode(rsax.CachedAutoDeterministic(t.seed, filepath.Join(t.credentialsDir, DefaultTLSKeyServer))); err != nil {
		return err
	}

	if template, err = tlsx.X509TemplateRand(rsax.NewSHA512CSPRNG(t.seed), minimumExpiration(), tlsx.DefaultClock(), subject, tlsx.X509OptionHosts(t.domain)); err != nil {
		return err
	}

	if _, cert, err = tlsx.SelfSigned(priv, &template); err != nil {
		return err
	}

	if envx.Boolean(false, bw.EnvLogsTLS, bw.EnvLogsVerbose) {
		log.Println("creating self signed certificate", tlsx.PrintEncoded(cert))
	}

	if err = tlsx.WriteCertificateFile(filepath.Join(t.credentialsDir, DefaultTLSBootstrapCert), cert); err != nil {
		return err
	}

	return nil
}
