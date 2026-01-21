package certificatecache

import (
	"context"
	"crypto/tls"
	"crypto/x509"
	"log"
	"os"
	"path/filepath"
	"time"

	"github.com/james-lawrence/bw"
	"github.com/james-lawrence/bw/agent"
	"github.com/james-lawrence/bw/agent/dialers"
	"github.com/james-lawrence/bw/internal/grpcx"
	"github.com/james-lawrence/bw/internal/tlsx"
	nsvc "github.com/james-lawrence/bw/notary"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"

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
	Insecure       bool
}

// Refresh the current credentials
func (t Notary) Refresh() (err error) {
	var (
		key  []byte
		cert []byte
		ca   []byte
		pool *x509.CertPool
		ss   nsvc.Signer
	)

	if ss, err = nsvc.NewAutoSigner(bw.DisplayName()); err != nil {
		return err
	}

	if pool, err = x509.SystemCertPool(); err != nil {
		log.Println(errors.Wrap(err, "WARN: unable to load certificate authorities, assuming static certificates"))
		return nil
	}

	if ca, err = os.ReadFile(t.CA); err != nil && !os.IsNotExist(err) {
		log.Println(errors.Wrap(err, "WARN: unable to load certificate authority, assuming static certificates"))
		return nil
	}

	if len(ca) > 0 {
		if ok := pool.AppendCertsFromPEM(ca); !ok {
			log.Println(errors.New("WARN: failed to append client certs, assuming static certificates"))
			return nil
		}
	}

	c := &tls.Config{
		ServerName:         t.CommonName,
		RootCAs:            pool,
		NextProtos:         []string{"bw.mux"},
		InsecureSkipVerify: t.Insecure,
	}

	d, err := dialers.DefaultDialer(t.Address, tlsx.NewDialer(c), grpc.WithPerRPCCredentials(ss))
	if err != nil {
		return err
	}

	dd := dialers.NewDirect(agent.URIAgent(t.Address), d.Defaults()...)
	client := nsvc.NewClient(dd)
	ctx, done := context.WithTimeout(context.Background(), time.Minute)
	defer done()

	err = grpcx.Retry(ctx, func() error {
		if ca, key, cert, err = client.Refresh(); err != nil {
			return err
		}
		return nil
	}, codes.Unavailable)
	if err != nil {
		return err
	}

	log.Println("refresh completed")
	capath := filepath.Join(t.CertificateDir, DefaultTLSCertCA)
	keypath := filepath.Join(t.CertificateDir, DefaultTLSKeyClient)
	certpath := filepath.Join(t.CertificateDir, DefaultTLSCertClient)

	log.Println("writing private key", keypath)
	if err = os.WriteFile(keypath, key, 0600); err != nil {
		return errors.Wrapf(err, "failed to write private key to %s", keypath)
	}

	log.Println("writing certificate", certpath)
	if err = os.WriteFile(certpath, cert, 0600); err != nil {
		return errors.Wrapf(err, "failed to write certificate to %s", certpath)
	}

	log.Println("writing authority certificate", capath)
	if err = os.WriteFile(capath, ca, 0600); err != nil {
		return errors.Wrapf(err, "failed to write certificate authority to %s", capath)
	}

	return nil
}
