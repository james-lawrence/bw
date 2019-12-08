package daemons

import (
	"bytes"
	"context"
	"crypto/rsa"
	"crypto/x509"
	"crypto/x509/pkix"
	"io/ioutil"
	"log"
	"time"

	"github.com/james-lawrence/bw/agent"
	"github.com/james-lawrence/bw/agent/quorum"
	"github.com/james-lawrence/bw/agentutil"
	"github.com/james-lawrence/bw/certificatecache"
	"github.com/james-lawrence/bw/internal/x/logx"
	"github.com/james-lawrence/bw/internal/x/systemx"
	"github.com/james-lawrence/bw/internal/x/tlsx"
)

// GenCertificate generate the cluster's certificate.
func GenCertificate(c agent.Config) quorum.Initializer {
	return genCA{c: c}
}

type genCA struct {
	c agent.Config
}

func (t genCA) Initialize(dispatch agent.Dispatcher) (err error) {
	var (
		cert      []byte
		capriv    *rsa.PrivateKey
		authority x509.Certificate
		keybuf    = bytes.NewBufferString("")
		certbuf   = bytes.NewBufferString("")
	)

	cakeypath := certificatecache.CAKeyPath(t.c.Root, certificatecache.DefaultTLSGeneratedCAKey)
	cacertpath := certificatecache.CACertPath(t.c.Root, certificatecache.DefaultTLSGeneratedCACert)
	log.Println("GENERATE CERTIFICATE AUTHORITY INITIATED", t.c.ServerName)
	defer func() {
		log.Println("GENERATE CERTIFICATE AUTHORITY COMPLETED")
		logx.MaybeLog(dispatch.Dispatch(context.Background(), agentutil.LogEvent(t.c.Peer(), "generate certificate authority completed")))
	}()

	if err = dispatch.Dispatch(context.Background(), agentutil.LogEvent(t.c.Peer(), "generate certificate authority initiated")); err != nil {
		return err
	}

	if systemx.FileExists(cakeypath) && systemx.FileExists(cacertpath) {
		log.Println("GENERATE CERTIFICATE AUTHORITY NOT NEEDED")
		return nil
	}

	caoptions := []tlsx.X509Option{
		tlsx.X509OptionCA(),
		tlsx.X509OptionSubject(pkix.Name{
			CommonName: t.c.ServerName,
		}),
	}

	if authority, err = tlsx.X509Template(8760*time.Hour, caoptions...); err != nil {
		return err
	}

	if capriv, cert, err = tlsx.SelfSignedRSAGen(4096, authority); err != nil {
		return err
	}

	if err = tlsx.WritePrivateKey(keybuf, capriv); err != nil {
		return err
	}

	if err = tlsx.WriteCertificate(certbuf, cert); err != nil {
		return err
	}

	// dispatch cert to cluster.
	if err = dispatch.Dispatch(context.Background(), agentutil.LogEvent(t.c.Peer(), "generate certificate authority completed")); err != nil {
		return err
	}

	if err = ioutil.WriteFile(cakeypath, keybuf.Bytes(), 0600); err != nil {
		return err
	}

	if err = ioutil.WriteFile(cacertpath, certbuf.Bytes(), 0600); err != nil {
		return err
	}

	return nil
}
