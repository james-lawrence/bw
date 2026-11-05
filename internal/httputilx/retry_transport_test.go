package httputilx_test

import (
	"context"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/james-lawrence/bw/internal/httptestx"

	. "github.com/james-lawrence/bw/internal/httputilx"

	"github.com/stretchr/testify/require"
)

func TestShouldRetryOnce(t *testing.T) {
	invoked := 0
	body := []byte("")
	c := httptestx.NewTestClient(func(req *http.Request) *http.Response {
		body, _ = io.ReadAll(req.Body)
		invoked++

		return &http.Response{
			StatusCode: http.StatusBadGateway,
			Body:       io.NopCloser(strings.NewReader("")),
			Header:     http.Header{},
		}
	})

	c.Transport = NewRetryTransport(c.Transport, http.StatusBadGateway)
	req, err := http.NewRequest(http.MethodGet, "http://example.com/", strings.NewReader("Hello World"))
	require.NoError(t, err)
	resp, err := c.Do(req)
	require.NoError(t, err)
	require.Equal(t, http.StatusBadGateway, resp.StatusCode)
	require.Equal(t, 2, invoked)
	require.Equal(t, []byte("Hello World"), body)
}

func TestShouldRetryOnContextdeadlineexceeded(t *testing.T) {
	s := httptest.NewServer(http.HandlerFunc(func(resp http.ResponseWriter, req *http.Request) {
		time.Sleep(time.Hour)
	}))
	defer s.Close()
	c := http.DefaultClient
	c.Transport = NewRetryTransport(c.Transport, http.StatusBadGateway)

	ctx, done := context.WithTimeout(context.Background(), -1*time.Second)
	defer done()
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, s.URL, strings.NewReader("Hello World"))
	require.NoError(t, err)
	resp, err := c.Do(req)
	require.Error(t, err)
	require.Nil(t, resp)
}

func TestShouldRetryWithANilBody(t *testing.T) {
	invoked := 0
	body := []byte("")
	c := httptestx.NewTestClient(func(req *http.Request) *http.Response {
		body, _ = io.ReadAll(req.Body)
		invoked++

		return &http.Response{
			StatusCode: http.StatusBadGateway,
			Body:       io.NopCloser(strings.NewReader("")),
			Header:     http.Header{},
		}
	})

	c.Transport = NewRetryTransport(c.Transport, http.StatusBadGateway)
	req, err := http.NewRequest(http.MethodGet, "http://example.com/", nil)
	require.NoError(t, err)
	resp, err := c.Do(req)
	require.NoError(t, err)
	require.Equal(t, http.StatusBadGateway, resp.StatusCode)
	require.Equal(t, 2, invoked)
	require.Equal(t, []byte(""), body)
}
