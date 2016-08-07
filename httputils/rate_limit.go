package httputils

import (
	"net/http"
	"time"
)

// RateLimit - Rate Limiting Round Tripper for http requests.
type RateLimit struct {
	*time.Ticker
	Delegate http.RoundTripper
}

// RoundTrip - See http.RoundTripper
func (t RateLimit) RoundTrip(req *http.Request) (*http.Response, error) {
	<-t.Ticker.C
	return t.Delegate.RoundTrip(req)
}
