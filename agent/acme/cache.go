package acme

import (
	"crypto/tls"
	"errors"
)

// ALPNCertCache an adapter that provides an alpn certificate cache for resolving
// challenges.
type ALPNCertCache struct{}

// GetCertificate returns a certificate based on the challenge.
func (t ALPNCertCache) GetCertificate(*tls.ClientHelloInfo) (*tls.Certificate, error) {
	return nil, errors.New("not implemented")
}
