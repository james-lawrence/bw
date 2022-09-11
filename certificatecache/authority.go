package certificatecache

import (
	"bytes"
	"crypto/rsa"
	"crypto/x509"
	"crypto/x509/pkix"
	"log"
	"os"
	"path/filepath"
	"sync"
	"time"

	"github.com/james-lawrence/bw/agentutil"
	"github.com/james-lawrence/bw/internal/rsax"
	"github.com/james-lawrence/bw/internal/x/errorsx"
	"github.com/james-lawrence/bw/internal/x/tlsx"
	"google.golang.org/protobuf/proto"
)

// ErrAuthorityNotAvailable ...
const ErrAuthorityNotAvailable = errorsx.String("authority not available")

// NewAuthorityCache cached authority from the directory.
func NewAuthorityCache(domain, dir string) *AuthorityCache {
	return &AuthorityCache{
		dir:    dir,
		domain: domain,
		m:      &sync.RWMutex{},
	}
}

// AuthorityCache lazy creation of the authority cache if possible.
type AuthorityCache struct {
	dir       string
	domain    string
	authority *Authority
	m         *sync.RWMutex
}

// Create certificate.
func (t *AuthorityCache) Create(duration time.Duration, bits int, options ...tlsx.X509Option) (ca, key, cert []byte, err error) {
	var (
		encoded  []byte
		template x509.Certificate
	)

	t.m.RLock()
	authority := t.authority
	t.m.RUnlock()

	if authority != nil && authority.CACert.NotAfter.After(time.Now()) {
		return authority.Create(duration, bits, options...)
	}

	t.m.Lock()
	defer t.m.Unlock()
	subject := tlsx.X509OptionSubject(pkix.Name{
		CommonName: t.domain,
	})

	authority = &Authority{}

	kpath := filepath.Join(t.dir, DefaultTLSKeyServer)
	certpath := filepath.Join(t.dir, DefaultDirTLSAuthority, DefaultTLSGeneratedCACert)
	protopath := filepath.Join(t.dir, DefaultTLSGeneratedCAProto)

	log.Println("loading ca key", kpath)
	log.Println("writing proto file", protopath)

	if authority.CAKey, err = rsax.DecodeFile(kpath); err != nil {
		log.Println("failed to initialize authority", err)
		return ca, key, cert, ErrAuthorityNotAvailable
	}

	if template, err = tlsx.X509TemplateRand(rsax.NewSHA512CSPRNG(nil), time.Minute, tlsx.DefaultClock(), subject, tlsx.X509OptionHosts(t.domain)); err != nil {
		return ca, key, cert, ErrAuthorityNotAvailable
	}

	if _, cert, err = tlsx.SelfSigned(authority.CAKey, &template); err != nil {
		log.Println("unable to create certificate", err)
		return ca, key, cert, ErrAuthorityNotAvailable
	}

	if err = tlsx.WriteCertificateFile(certpath, cert); err != nil {
		return ca, key, cert, ErrAuthorityNotAvailable
	}

	if authority.CACert, err = x509.ParseCertificate(cert); err != nil {
		log.Println("unable to decode certificate", err)
		return ca, key, cert, ErrAuthorityNotAvailable
	}

	prototlscert := agentutil.TLSCertificates(cert, cert)
	if encoded, err = proto.Marshal(&prototlscert); err != nil {
		return ca, key, cert, ErrAuthorityNotAvailable
	}

	if err = os.WriteFile(protopath, encoded, 0600); err != nil {
		return ca, key, cert, ErrAuthorityNotAvailable
	}

	t.authority = authority

	return t.authority.Create(duration, bits, options...)
}

// Authority used to generate certificates.
type Authority struct {
	CACert *x509.Certificate
	CAKey  *rsa.PrivateKey
}

// Create certificate from options
func (t Authority) Create(duration time.Duration, bits int, options ...tlsx.X509Option) (ca, key, cert []byte, err error) {
	var (
		template  x509.Certificate
		generated *rsa.PrivateKey
		keybuf    = bytes.NewBufferString("")
		certbuf   = bytes.NewBufferString("")
		cabuf     = bytes.NewBufferString("")
	)

	if template, err = tlsx.X509Template(duration, options...); err != nil {
		return ca, key, cert, err
	}

	if generated, cert, err = tlsx.SignedRSAGen(bits, template, *t.CACert, t.CAKey); err != nil {
		return ca, key, cert, err
	}

	if err = tlsx.WritePrivateKey(keybuf, generated); err != nil {
		return ca, key, cert, err
	}

	if err = tlsx.WriteCertificate(certbuf, cert); err != nil {
		return ca, key, cert, err
	}

	if err = tlsx.WriteCertificate(cabuf, t.CACert.Raw); err != nil {
		return ca, key, cert, err
	}

	return cabuf.Bytes(), keybuf.Bytes(), certbuf.Bytes(), err
}
