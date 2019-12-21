package certificatecache

import (
	"crypto/rsa"
	"crypto/x509"
	"crypto/x509/pkix"
	"path/filepath"
	"time"

	"github.com/james-lawrence/bw/internal/x/tlsx"
)

// generates a self signed certificate iff the current certificate is missing or
// expired. this is used to allow the cluster to bootstrap correctly.
type selfsigned struct {
	domain         string
	credentialsDir string
}

func (t selfsigned) Refresh() (err error) {
	var (
		priv     *rsa.PrivateKey
		cert     []byte
		template x509.Certificate
	)

	subject := tlsx.X509OptionSubject(pkix.Name{
		CommonName: t.domain,
	})

	if template, err = tlsx.X509Template(10*time.Minute, subject); err != nil {
		return err
	}

	if priv, cert, err = tlsx.SelfSignedRSAGen(8096, template); err != nil {
		return err
	}

	if err = tlsx.WritePrivateKeyFile(filepath.Join(t.credentialsDir, DefaultTLSKeyServer), priv); err != nil {
		return err
	}

	if err = tlsx.WriteCertificateFile(filepath.Join(t.credentialsDir, DefaultTLSCertServer), cert); err != nil {
		return err
	}

	return nil
}
