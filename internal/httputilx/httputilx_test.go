package httputilx_test

import (
	"crypto/tls"
	"fmt"
	"net"
	"net/http"
	"net/url"
	"testing"

	. "github.com/james-lawrence/bw/internal/httputilx"

	"github.com/stretchr/testify/require"
)

func TestIsWebsocketShutdownError(t *testing.T) {
	for _, c := range []struct {
		name   string
		err    error
		result bool
	}{
		{"unknown error", fmt.Errorf("unknown error"), false},
		{"tls closed connection", fmt.Errorf("tls: use of closed connection"), true},
		{"temporary error should return false", temporaryError{temp: true, err: fmt.Errorf("temporary error")}, false},
		{"permenant error should return true", temporaryError{temp: false, err: fmt.Errorf("permanent error")}, true},
	} {
		t.Run(c.name, func(t *testing.T) {
			require.Equal(t, c.result, IsWebsocketShutdownError(c.err))
		})
	}
}

func TestRedirectHTTPRequest(t *testing.T) {
	for _, c := range []struct {
		name        string
		inURL       *url.URL
		cs          *tls.ConnectionState
		inIP        net.IP
		expectedURL string
		defaultPort string
	}{
		{
			"it should use the port provided in the host field",
			&url.URL{Scheme: "http", Host: "www.example.com:123", Path: "tallachat/details"},
			nil,
			net.ParseIP("127.0.0.1"),
			"http://127.0.0.1:123/tallachat/details",
			"456",
		},
		{
			"it should use scheme provided by the url",
			&url.URL{Scheme: "https", Host: "www.example.com:123", Path: "tallachat/details"},
			&tls.ConnectionState{},
			net.ParseIP("127.0.0.1"),
			"https://127.0.0.1:123/tallachat/details",
			"456",
		},
		{
			"it should use the default port if no port is provided in the host field",
			&url.URL{Scheme: "http", Host: "www.example.com", Path: "tallachat/details"},
			nil,
			net.ParseIP("127.0.0.1"),
			"http://127.0.0.1:456/tallachat/details",
			"456",
		},
	} {
		t.Run(c.name, func(t *testing.T) {
			req := &http.Request{Host: c.inURL.Host, URL: c.inURL, TLS: c.cs}
			require.Equal(t, c.expectedURL, RedirectHTTPRequest(req, c.inIP.String(), c.defaultPort).String())
		})
	}
}

type temporaryError struct {
	temp bool
	err  error
}

func (t temporaryError) Error() string {
	return t.err.Error()
}

func (t temporaryError) Temporary() bool {
	return t.temp
}
