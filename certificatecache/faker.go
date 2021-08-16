package certificatecache

import (
	"crypto/rsa"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/pem"
	"io/ioutil"
	"log"
	"path/filepath"
	"time"

	"github.com/james-lawrence/bw/internal/rsax"
	"github.com/james-lawrence/bw/internal/x/tlsx"
	"github.com/pkg/errors"
)

// generates a self signed certificate iff the current certificate is missing or
// expired. this is used to allow the cluster to bootstrap correctly.
type faker struct {
	seed           []byte
	domain         string
	CertificateDir string
}

func (t faker) Refresh() (err error) {
	var (
		priv     *rsa.PrivateKey
		cert     []byte
		template x509.Certificate
	)

	subject := tlsx.X509OptionSubject(pkix.Name{
		CommonName: t.domain,
	})

	if priv, err = rsax.MaybeDecode(rsax.CachedAutoDeterministic(t.seed, filepath.Join(t.CertificateDir, DefaultTLSKeyServer))); err != nil {
		return err
	}

	if template, err = tlsx.X509Template(30*24*time.Hour, subject, tlsx.X509OptionCA(), tlsx.X509OptionHosts(t.domain)); err != nil {
		return err
	}

	if _, cert, err = tlsx.SelfSigned(priv, template); err != nil {
		return err
	}

	if cert = pem.EncodeToMemory(&pem.Block{Type: "CERTIFICATE", Bytes: cert}); err != nil {
		return errors.WithStack(err)
	}

	capath := filepath.Join(t.CertificateDir, DefaultTLSCertCA)
	certpath := filepath.Join(t.CertificateDir, DefaultTLSCertServer)

	log.Println("writing certificate", certpath)
	if err = ioutil.WriteFile(certpath, cert, 0600); err != nil {
		return errors.Wrapf(err, "failed to write certificate to %s", certpath)
	}

	log.Println("writing authority certificate", capath)
	if err = ioutil.WriteFile(capath, cert, 0600); err != nil {
		return errors.Wrapf(err, "failed to write certificate authority to %s", capath)
	}

	return nil
}
