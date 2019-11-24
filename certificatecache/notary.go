package certificatecache

import (
	"crypto/tls"
	"crypto/x509"
	"io/ioutil"
	"log"
	"os"
	"path/filepath"

	"github.com/james-lawrence/bw/agent/discovery"
	nsvc "github.com/james-lawrence/bw/notary"
	"github.com/pkg/errors"
)

// notary configuration used for bootstrapping a client from the cluster itself.
// for legacy reasons notary system will return an error if the cluster doesn't support
// the notary service.
type notary struct {
	Address        string `yaml:"address"`
	CommonName     string `yaml:"servername"`
	CertificateDir string
	CA             string
}

func (t notary) Refresh() (err error) {
	var (
		key  []byte
		cert []byte
		ca   []byte
		pool *x509.CertPool
	)

	if pool, err = x509.SystemCertPool(); err != nil {
		log.Println(errors.Wrap(err, "WARN: unable to load certificate authorities, assuming static certificates"))
		return nil
	}

	if ca, err = ioutil.ReadFile(t.CA); err != nil && !os.IsNotExist(err) {
		log.Println(errors.Wrap(err, "WARN: unable to load certificate authority, assuming static certificates"))
		return nil
	}

	if len(ca) > 0 {
		if ok := pool.AppendCertsFromPEM(ca); !ok {
			log.Println(errors.New("WARN: failed to append client certs, assuming static certificates"))
			return nil
		}
	}

	c := tls.Config{
		ServerName: t.CommonName,
		RootCAs:    pool,
	}

	client := nsvc.NewClient(nsvc.NewDialer(discovery.NewQuorumDialer(t.Address), nsvc.DialOptionTLS(&c)))

	if key, cert, err = client.Refresh(); err != nil {
		return err
	}

	// capath := filepath.Join(t.CertificateDir, DefaultTLSCertCA)
	keypath := filepath.Join(t.CertificateDir, DefaultTLSKeyServer)
	certpath := filepath.Join(t.CertificateDir, DefaultTLSCertServer)

	log.Println("writing private key", keypath)
	if err = ioutil.WriteFile(keypath, key, 0600); err != nil {
		return errors.Wrapf(err, "failed to write private key to %s", keypath)
	}

	log.Println("writing certificate", certpath)
	if err = ioutil.WriteFile(certpath, cert, 0600); err != nil {
		return errors.Wrapf(err, "failed to write certificate to %s", certpath)
	}

	// log.Println("writing authority certificate", capath)
	// if err = ioutil.WriteFile(capath, []byte(credentials.Data["issuing_ca"].(string)), 0600); err != nil {
	// 	return errors.Wrapf(err, "failed to write certificate authority to %s", capath)
	// }

	return nil
}
