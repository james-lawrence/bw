package certificatecache

import (
	"crypto/tls"
	"crypto/x509"
	"io/ioutil"
	"log"
	"os"
	"path/filepath"

	"github.com/james-lawrence/bw"
	"github.com/james-lawrence/bw/agent/discovery"
	"github.com/james-lawrence/bw/internal/x/grpcx"
	nsvc "github.com/james-lawrence/bw/notary"

	"github.com/pkg/errors"
)

// Notary refresher used for bootstrapping a client from the cluster itself.
// for legacy reasons notary system will return an error if the cluster doesn't support
// the notary service.
type Notary struct {
	Address        string `yaml:"address"`
	Discovery      string `yaml:"discovery"`
	CommonName     string `yaml:"servername"`
	CertificateDir string
	CA             string
}

// Refresh the current credentials
func (t Notary) Refresh() (err error) {
	var (
		key  []byte
		cert []byte
		ca   []byte
		pool *x509.CertPool
		d    discovery.QuorumDialer
		ss   nsvc.Signer
	)

	if ss, err = nsvc.NewAutoSigner(bw.DisplayName()); err != nil {
		return err
	}

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

	log.Println("dialing discovery service", t.CommonName, t.Discovery)
	if d, err = discovery.NewQuorumDialer(t.Discovery); err != nil {
		return err
	}

	client := nsvc.NewClient(nsvc.NewDialer(d, nsvc.DialOptionTLS(&c), nsvc.DialOptionCredentials(ss)))

	if ca, key, cert, err = client.Refresh(); err != nil {
		// backwards compatibility code, for now only consider permission errors
		// as hard failures, not all agents have the discovery service.
		if grpcx.IsUnauthorized(err) {
			return nsvc.ErrUnauthorizedKey{}
		}

		// backwards compatibility code.
		return Noop{}.Refresh()
	}

	log.Println("refresh completed")
	capath := filepath.Join(t.CertificateDir, DefaultTLSCertCA)
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

	log.Println("writing authority certificate", capath)
	if err = ioutil.WriteFile(capath, ca, 0600); err != nil {
		return errors.Wrapf(err, "failed to write certificate authority to %s", capath)
	}

	return nil
}
