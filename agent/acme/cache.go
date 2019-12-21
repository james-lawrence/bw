package acme

import (
	"context"
	"crypto/tls"
	"errors"
	"log"
)

type resolution interface {
	Resolution(ctx context.Context) (c Challenge, err error)
}

// NewALPNCertCache certificate lookup for ALPN requests.
func NewALPNCertCache(r resolution) ALPNCertCache {
	return ALPNCertCache{r: r}
}

// ALPNCertCache an adapter that provides an alpn certificate cache for resolving
// challenges.
type ALPNCertCache struct {
	r resolution
}

// GetCertificate returns a certificate based on the challenge.
func (t ALPNCertCache) GetCertificate(hello *tls.ClientHelloInfo) (cert *tls.Certificate, err error) {
	// https://github.com/caddyserver/caddy/pull/2201/files
	log.Println("$$$$$$$$$$$$$$$$$$$$$ ACME DETECTED", hello.SupportedProtos)
	return nil, errors.New("not implemented")
}
