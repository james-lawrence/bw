package certificatecache

import (
	"bytes"
	"crypto/rsa"
	"crypto/x509"
	"encoding/pem"
	"io/ioutil"
	"log"
	"path/filepath"
	"sync"
	"time"

	"github.com/james-lawrence/bw/internal/x/errorsx"
	"github.com/james-lawrence/bw/internal/x/tlsx"
)

const ErrAuthorityNotAvailable = errorsx.String("authority not available")

// NewAuthorityCache cached authority from the directory.
func NewAuthorityCache(dir string) *AuthorityCache {
	return &AuthorityCache{
		dir: dir,
		m:   &sync.RWMutex{},
	}
}

// AuthorityCache lazy creation of the authority cache if possible.
type AuthorityCache struct {
	dir       string
	authority *Authority
	m         *sync.RWMutex
}

// Create certificate.
func (t *AuthorityCache) Create(duration time.Duration, bits int, options ...tlsx.X509Option) (ca, key, cert []byte, err error) {
	var (
		raw []byte
		b   *pem.Block
	)

	t.m.RLock()
	authority := t.authority
	t.m.RUnlock()

	if authority != nil {
		return authority.Create(duration, bits, options...)
	}

	t.m.Lock()
	defer t.m.Unlock()
	t.authority = &Authority{}

	kpath := filepath.Join(t.dir, DefaultTLSGeneratedCAKey)
	cpath := filepath.Join(t.dir, DefaultDirTLSAuthority, DefaultTLSGeneratedCACert)

	log.Println("loading ca key", kpath)
	log.Println("loading ca cert", cpath)

	if raw, err = ioutil.ReadFile(kpath); err != nil {
		log.Println("failed to initialize authority", err)
		return ca, key, cert, ErrAuthorityNotAvailable
	}

	if b, _ = pem.Decode(raw); err != nil {
		log.Println("failed to initialize authority", err)
		return ca, key, cert, ErrAuthorityNotAvailable
	}

	if t.authority.CAKey, err = x509.ParsePKCS1PrivateKey(b.Bytes); err != nil {
		log.Println("failed to initialize authority", err)
		return ca, key, cert, ErrAuthorityNotAvailable
	}

	if raw, err = ioutil.ReadFile(cpath); err != nil {
		log.Println("failed to initialize authority", err)
		return ca, key, cert, ErrAuthorityNotAvailable
	}

	if b, _ = pem.Decode(raw); err != nil {
		log.Println("failed to initialize authority", err)
		return ca, key, cert, ErrAuthorityNotAvailable
	}

	if t.authority.CACert, err = x509.ParseCertificate(b.Bytes); err != nil {
		log.Println("failed to initialize authority", err)
		return ca, key, cert, ErrAuthorityNotAvailable
	}

	log.Println("creating credentials")
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

	log.Println("checkpoint 1")
	if template, err = tlsx.X509Template(duration, options...); err != nil {
		return ca, key, cert, err
	}
	log.Println("checkpoint 2")
	if generated, cert, err = tlsx.SignedRSAGen(bits, template, *t.CACert, t.CAKey); err != nil {
		return ca, key, cert, err
	}
	log.Println("checkpoint 3")
	if err = tlsx.WritePrivateKey(keybuf, generated); err != nil {
		return ca, key, cert, err
	}
	log.Println("checkpoint 4")
	if err = tlsx.WriteCertificate(certbuf, cert); err != nil {
		return ca, key, cert, err
	}
	log.Println("checkpoint 5")
	if err = tlsx.WriteCertificate(cabuf, t.CACert.Raw); err != nil {
		return ca, key, cert, err
	}
	log.Println("checkpoint 6")
	return cabuf.Bytes(), keybuf.Bytes(), certbuf.Bytes(), err
}
