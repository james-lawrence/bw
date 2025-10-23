package acme

import (
	"context"
	"crypto/tls"

	"github.com/go-acme/lego/v4/challenge/tlsalpn01"
	"github.com/james-lawrence/bw/internal/errorsx"
	"github.com/pkg/errors"
)

type resolution interface {
	Resolution(ctx context.Context) (c *Challenge, err error)
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
func (t ALPNCertCache) GetCertificate(_ *tls.ClientHelloInfo) (cert *tls.Certificate, err error) {
	var (
		cc *Challenge
	)

	if cc, err = t.r.Resolution(context.Background()); err != nil {
		cause := errors.Wrap(err, "failed to retrieve challenge")
		errorsx.Log(cause)
		return nil, cause
	}

	return tlsalpn01.ChallengeCert(cc.Domain, cc.Digest)
}
