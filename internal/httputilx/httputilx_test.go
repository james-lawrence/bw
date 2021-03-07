package httputilx_test

import (
	"crypto/tls"
	"fmt"
	"net"
	"net/http"
	"net/url"

	. "github.com/james-lawrence/bw/internal/httputilx"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/ginkgo/extensions/table"
	. "github.com/onsi/gomega"
)

var _ = Describe("HTTPutilx", func() {
	DescribeTable("IsWebsocketShutdownError",
		func(err error, result bool) {
			Expect(IsWebsocketShutdownError(err)).To(Equal(result))
		},
		Entry("unknown error", fmt.Errorf("unknown error"), false),
		Entry("tls closed connection", fmt.Errorf("tls: use of closed connection"), true),
		Entry("temporary error should return false", temporaryError{temp: true, err: fmt.Errorf("temporary error")}, false),
		Entry("permenant error should return true", temporaryError{temp: false, err: fmt.Errorf("permanent error")}, true),
	)

	DescribeTable("RedirectHTTPRequest",
		func(inURL *url.URL, cs *tls.ConnectionState, inIP net.IP, expectedURL, defaultPort string) {
			req := &http.Request{Host: inURL.Host, URL: inURL, TLS: cs}
			Expect(RedirectHTTPRequest(req, inIP.String(), defaultPort).String()).To(Equal(expectedURL))
		},
		Entry(
			"it should use the port provided in the host field",
			&url.URL{Scheme: "http", Host: "www.example.com:123", Path: "tallachat/details"},
			nil,
			net.ParseIP("127.0.0.1"),
			"http://127.0.0.1:123/tallachat/details",
			"456",
		),
		Entry(
			"it should use scheme provided by the url",
			&url.URL{Scheme: "https", Host: "www.example.com:123", Path: "tallachat/details"},
			&tls.ConnectionState{},
			net.ParseIP("127.0.0.1"),
			"https://127.0.0.1:123/tallachat/details",
			"456",
		),
		Entry(
			"it should use the default port if no port is provided in the host field",
			&url.URL{Scheme: "http", Host: "www.example.com", Path: "tallachat/details"},
			nil,
			net.ParseIP("127.0.0.1"),
			"http://127.0.0.1:456/tallachat/details",
			"456",
		),
	)
})

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
