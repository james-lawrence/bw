package certificatecache

import (
	"crypto/tls"

	"github.com/go-acme/lego/v4/challenge/tlsalpn01"
)

// represents a certificate cache
type cache interface {
	GetCertificate(*tls.ClientHelloInfo) (*tls.Certificate, error)
}

// NewALPN clones the provided TLS config and updates the GetCertificate method
func NewALPN(c *tls.Config, cc cache) *tls.Config {
	updated := c.Clone()
	updated.NextProtos = append(updated.NextProtos, tlsalpn01.ACMETLS1Protocol)
	updated.GetCertificate = ALPN{cache: cc, fallback: c.GetCertificate}.GetCertificate
	return updated
}

// ALPN implements the alpn TLS certificate resolution strategy.
type ALPN struct {
	cache
	fallback func(*tls.ClientHelloInfo) (*tls.Certificate, error)
}

// GetCertificate for use by tls.Config.
func (t ALPN) GetCertificate(hello *tls.ClientHelloInfo) (*tls.Certificate, error) {
	for _, proto := range hello.SupportedProtos {
		if proto == tlsalpn01.ACMETLS1Protocol {
			return t.cache.GetCertificate(hello)
		}
	}

	if t.fallback == nil {
		return nil, nil
	}

	return t.fallback(hello)
}
