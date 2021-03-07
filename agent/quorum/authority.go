package quorum

import (
	"bytes"
	"context"
	"crypto/rsa"
	"crypto/x509"
	"crypto/x509/pkix"
	"io"
	"io/ioutil"
	"log"
	"os"
	"path/filepath"
	"time"

	"github.com/golang/protobuf/proto"
	"github.com/james-lawrence/bw/agent"
	"github.com/james-lawrence/bw/agentutil"
	"github.com/james-lawrence/bw/certificatecache"
	"github.com/james-lawrence/bw/internal/x/systemx"
	"github.com/james-lawrence/bw/internal/x/tlsx"
)

// NewAuthority from agent configuration.
func NewAuthority(c agent.Config) Authority {
	return Authority{
		c:           c,
		pathCAProto: certificatecache.CAKeyPath(c.CredentialsDir, certificatecache.DefaultTLSGeneratedCAProto),
		pathCAKey:   certificatecache.CAKeyPath(c.CredentialsDir, certificatecache.DefaultTLSGeneratedCAKey),
		pathCACert:  certificatecache.CACertPath(c.CredentialsDir, certificatecache.DefaultTLSGeneratedCACert),
	}
}

// Authority ...
type Authority struct {
	c           agent.Config
	pathCAProto string
	pathCAKey   string
	pathCACert  string
}

// Encode the authority proto message into the event.
func (t Authority) Encode(dst io.Writer) (err error) {
	var (
		buf []byte
	)

	if buf, err = ioutil.ReadFile(t.pathCAProto); err != nil {
		return err
	}

	if _, err = dst.Write(encodeRaw(buf)); err != nil {
		return err
	}

	return nil
}

// Decode ...
func (t Authority) Decode(_ TranscoderContext, m *agent.Message) (err error) {
	var (
		buf []byte
		evt *agent.TLSEvent
	)

	switch event := m.GetEvent().(type) {
	case *agent.Message_Authority:
		evt = event.Authority
	default:
		return nil
	}

	log.Println("consume generated tls initiated", evt.Fingerprint)
	defer log.Println("consume generated tls completed", evt.Fingerprint)

	if err = t.write(evt); err != nil {
		return nil
	}

	if buf, err = proto.Marshal(m); err != nil {
		return err
	}

	if err = ioutil.WriteFile(t.pathCAProto, buf, 0600); err != nil {
		return err
	}

	return nil
}

// Initialize the authority.
func (t Authority) Initialize(dispatch agent.Dispatcher) (err error) {
	if systemx.FileExists(t.pathCAProto) {
		return t.load(dispatch)
	}

	return t.generate(dispatch)
}

func (t Authority) load(d agent.Dispatcher) (err error) {
	var (
		m       agent.Message
		encoded []byte
	)

	if encoded, err = ioutil.ReadFile(t.pathCAProto); err != nil {
		return err
	}

	if err = proto.Unmarshal(encoded, &m); err != nil {
		return err
	}

	return d.Dispatch(context.Background(), &m)
}

func (t Authority) generate(d agent.Dispatcher) (err error) {
	var (
		cert      []byte
		capriv    *rsa.PrivateKey
		authority x509.Certificate
		keybuf    = bytes.NewBufferString("")
		certbuf   = bytes.NewBufferString("")
	)

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
	return d.Dispatch(context.Background(), agentutil.TLSEventMessage(t.c.Peer(), keybuf.Bytes(), certbuf.Bytes()))
}

func (t Authority) write(evt *agent.TLSEvent) (err error) {
	if err = os.MkdirAll(filepath.Dir(t.pathCACert), 0700); err != nil {
		return err
	}

	if err = ioutil.WriteFile(t.pathCACert, evt.Certificate, 0600); err != nil {
		return err
	}

	if err = ioutil.WriteFile(t.pathCAKey, evt.Key, 0600); err != nil {
		return err
	}

	return nil
}

func (t Authority) read() (evt agent.TLSEvent, err error) {
	var (
		cert []byte
		key  []byte
	)

	if cert, err = ioutil.ReadFile(t.pathCACert); err != nil {
		return evt, err
	}

	if key, err = ioutil.ReadFile(t.pathCAKey); err != nil {
		return evt, err
	}

	return agentutil.TLSEvent(key, cert), nil
}
