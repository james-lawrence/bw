package httputilx

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"mime"
	"net"
	"net/http"
	"net/http/httputil"
	"net/url"
	"os"
	"strconv"
	"strings"
	"sync/atomic"
	"time"

	"github.com/gorilla/mux"
	"github.com/gorilla/websocket"
	"github.com/hashicorp/memberlist"
	"github.com/james-lawrence/bw/backoff"
	"github.com/james-lawrence/bw/internal/debugx"
	"github.com/james-lawrence/bw/internal/stringsx"
	"github.com/pkg/errors"
	"golang.org/x/time/rate"
)

// CheckStatusCode compares the provided status code with a list of acceptable
// status codes.
func CheckStatusCode(actual int, acceptable ...int) bool {
	for _, code := range acceptable {
		if actual == code {
			return true
		}
	}

	return false
}

// IsSuccess - returns true iff the response code was one of the following:
// http.StatusOK, http.StatusAccepted, http.StatusCreated. Delegates to CheckStatusCode, http.StatusNoContent.
func IsSuccess(actual int) bool {
	return CheckStatusCode(actual, http.StatusNoContent, http.StatusOK, http.StatusAccepted, http.StatusCreated)
}

// Get return a get request for the given endpoint
func Get(ctx context.Context, endpoint string) (*http.Request, error) {
	return http.NewRequestWithContext(ctx, http.MethodGet, endpoint, strings.NewReader(""))
}

// ParseFormHandler automatically triggers a parse of the request form.
func ParseFormHandler(original http.Handler) http.Handler {
	return http.HandlerFunc(func(resp http.ResponseWriter, req *http.Request) {
		if err := req.ParseForm(); err != nil {
			http.Error(resp, "malformatted form", http.StatusBadRequest)
			return
		}

		original.ServeHTTP(resp, req)
	})
}

// RouteInvokedHandler wraps a http.Handler and emits route invocations.
func RouteInvokedHandler(original http.Handler) http.Handler {
	return http.HandlerFunc(func(resp http.ResponseWriter, req *http.Request) {
		p := req.Host + req.URL.Path
		if route := mux.CurrentRoute(req); route != nil && len(route.GetName()) > 0 {
			p = route.GetName()
		}
		started := time.Now()
		log.Println(p, "invoked")
		original.ServeHTTP(resp, req)
		log.Println(p, "completed", time.Since(started))
	})
}

// RouteRateLimited applies a rate limit to the http handler.
func RouteRateLimited(l *rate.Limiter) func(http.Handler) http.Handler {
	return func(original http.Handler) http.Handler {
		attempts := int64(0)
		b := backoff.New(
			backoff.Exponential(32*time.Millisecond),
			backoff.Maximum(2*time.Second),
		)

		return http.HandlerFunc(func(resp http.ResponseWriter, req *http.Request) {
			if l.Allow() {
				atomic.StoreInt64(&attempts, 0)
				original.ServeHTTP(resp, req)
				return
			}

			nattempt := int(atomic.AddInt64(&attempts, 1))
			resp.Header().Add("Retry-After", fmt.Sprintf("%d", int(b.Backoff(nattempt)/time.Second)))
			resp.WriteHeader(http.StatusTooManyRequests)
		})
	}
}

// DumpRequestHandler dumps the request to STDERR.
func DumpRequestHandler(original http.Handler) http.Handler {
	return http.HandlerFunc(func(resp http.ResponseWriter, req *http.Request) {
		raw, err := httputil.DumpRequest(req, true)
		if err != nil {
			log.Println(errors.Wrap(err, "failed to dump request"))
		} else {
			log.Println(string(raw))
		}
		original.ServeHTTP(resp, req)
	})
}

// RecordRequestHandler records the request to a temporary file.
// does not clean up the file from disk.
func RecordRequestHandler(original http.Handler) http.Handler {
	return http.HandlerFunc(func(resp http.ResponseWriter, req *http.Request) {
		var (
			err error
			raw []byte
			out *os.File
		)

		if raw, err = httputil.DumpRequest(req, true); err != nil {
			log.Println("failed to dump request", err)
			goto next
		}

		if out, err = os.CreateTemp("", "request-recording"); err != nil {
			log.Println("failed to record request", err)
			goto next
		}
		defer out.Close()

		if _, err = out.Write(raw); err != nil {
			log.Println("failed to record contents to file", err)
			goto next
		}
	next:
		original.ServeHTTP(resp, req)
	})
}

// HTTPRequestScheme return the http scheme for a request.
func HTTPRequestScheme(req *http.Request) string {
	const (
		scheme       = "http"
		secureScheme = "https"
	)

	if req.TLS == nil {
		return scheme
	}

	return secureScheme
}

// WebsocketRequestScheme return the websocket scheme for a request.
func WebsocketRequestScheme(req *http.Request) string {
	const (
		scheme       = "ws"
		secureScheme = "wss"
	)

	if req.TLS == nil {
		return scheme
	}

	return secureScheme
}

// WriteJSON writes a json payload into the provided buffer and sets the context-type header to application/json.
func WriteJSON(resp http.ResponseWriter, buffer *bytes.Buffer, context interface{}) error {
	var (
		err error
	)

	buffer.Reset()
	resp.Header().Set("Content-Type", "application/json")

	if err = json.NewEncoder(buffer).Encode(context); err != nil {
		resp.WriteHeader(http.StatusInternalServerError)
		return err
	}

	_, err = io.Copy(resp, buffer)
	return err
}

// WriteEmptyJSONArray emits an empty json array with the provided status code.
func WriteEmptyJSONArray(resp http.ResponseWriter, code int) {
	const emptyJSON = "[]"
	resp.Header().Set("Content-Type", "application/json")
	resp.WriteHeader(code)
	resp.Write([]byte(emptyJSON))
}

// WriteEmptyJSON emits empty json with the provided status code.
func WriteEmptyJSON(resp http.ResponseWriter, code int) {
	const emptyJSON = "{}"
	resp.Header().Set("Content-Type", "application/json")
	resp.WriteHeader(code)
	resp.Write([]byte(emptyJSON))
}

// IsWebsocketShutdownError detects shutdown errors for websocket connections.
func IsWebsocketShutdownError(err error) bool {
	type temporary interface {
		error
		Temporary() bool
	}

	// check sentinal values.
	switch err {
	case websocket.ErrCloseSent:
		return true
	}

	switch r := err.(type) {
	case *websocket.CloseError:
		return true
	case temporary:
		return !r.Temporary()
	default:
		switch r.Error() {
		case "tls: use of closed connection":
			return true
		}
	}

	return false
}

// RedirectHTTPRequest generates a url to redirect to from the provided
// request and destination node
func RedirectHTTPRequest(req *http.Request, dst string, defaultPort string) *url.URL {
	_, port, err := net.SplitHostPort(req.Host)
	if err != nil {
		debugx.Println("using default port error splitting request host", err)
		port = defaultPort
	}

	return &url.URL{
		Scheme:   HTTPRequestScheme(req),
		Host:     net.JoinHostPort(dst, port),
		Path:     req.URL.Path,
		RawQuery: req.URL.Query().Encode(),
	}
}

type clusterreroute interface {
	IsLocal([]byte) (bool, *memberlist.Node)
}

// Redirect a request to a particular server.
func Redirect(streamID string, c clusterreroute, defaultPort string, req *http.Request, resp http.ResponseWriter) bool {
	// can safely ignore the error during parsing. it'll default to false.
	if ignoreRedirect, _ := strconv.ParseBool(req.FormValue("ignore-clustering-redirect")); ignoreRedirect {
		return false
	}

	local, dst := c.IsLocal([]byte(streamID))
	if local {
		debugx.Println("accepting request on local node")
		return false
	}

	location := RedirectHTTPRequest(req, dst.Addr.String(), stringsx.DefaultIfBlank(req.Header.Get("X-Forwarded-Port"), defaultPort))
	debugx.Println("redirecting to correct node", location.String())
	resp.Header().Add("Location", location.String())
	resp.WriteHeader(http.StatusMovedPermanently)
	return true
}

// PublicRedirect a request to a particular server.
func PublicRedirect(streamID string, c clusterreroute, defaultPort string, req *http.Request, resp http.ResponseWriter) bool {
	// can safely ignore the error during parsing. it'll default to false.
	if ignoreRedirect, _ := strconv.ParseBool(req.FormValue("ignore-clustering-redirect")); ignoreRedirect {
		return false
	}

	local, dst := c.IsLocal([]byte(streamID))
	if local {
		debugx.Println("accepting request on local node")
		return false
	}

	location := RedirectHTTPRequest(req, dst.Name, stringsx.DefaultIfBlank(req.Header.Get("X-Forwarded-Port"), defaultPort))
	debugx.Println("redirecting to correct node", location.String())
	resp.Header().Add("Location", location.String())
	resp.WriteHeader(http.StatusMovedPermanently)
	return true
}

// ErrorCode ...
func ErrorCode(resp *http.Response) error {
	if resp.StatusCode < 400 {
		return nil
	}

	return Error{Code: resp.StatusCode, cause: errors.New(resp.Status)}
}

// Error ...
type Error struct {
	Code  int
	cause error
}

func (t Error) Error() string {
	return t.cause.Error()
}

// IgnoreError ...
func IgnoreError(err error, code ...int) bool {
	var (
		cause Error
		ok    bool
	)

	if cause, ok = errors.Cause(err).(Error); !ok {
		return false
	}

	return CheckStatusCode(cause.Code, code...)
}

// MimeType extracts mimetype from request, defaults to application/
func MimeType(h http.Header) string {
	const fallback = "application/octet-stream"
	t, _, err := mime.ParseMediaType(h.Get("Content-Type"))
	if err != nil {
		return fallback
	}

	return stringsx.DefaultIfBlank(t, fallback)
}

// Notfound response handler
func NotFound(resp http.ResponseWriter, req *http.Request) {
	raw, _ := httputil.DumpRequest(req, false)
	log.Println("requested endpoint not found", string(raw))
	resp.WriteHeader(http.StatusNotFound)
}
