package discovery

import (
	"context"
	"errors"
	"io"
	"net"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

type allow struct{ token string }

func (t allow) Authorization(b []byte) error {
	if string(b) != t.token {
		return errors.New("unauthorized")
	}
	return nil
}

type signed string

func (t signed) Token() (string, error) { return string(t), nil }

type netdialer struct{}

func (netdialer) DialContext(ctx context.Context, network, address string) (net.Conn, error) {
	return (&net.Dialer{}).DialContext(ctx, network, address)
}

// failing dialer that records what it was asked to connect to.
type refuse struct{ addr chan string }

func (t refuse) DialContext(ctx context.Context, network, address string) (net.Conn, error) {
	t.addr <- address
	return nil, errors.New("refused")
}

func echoServer(t *testing.T) net.Listener {
	l, err := net.Listen("tcp", "127.0.0.1:0")
	require.NoError(t, err)
	t.Cleanup(func() { l.Close() })
	go func() {
		for {
			c, err := l.Accept()
			if err != nil {
				return
			}
			go func() { defer c.Close(); io.Copy(c, c) }()
		}
	}()
	return l
}

func proxyServer(t *testing.T, d dialer, a auth) net.Listener {
	l, err := net.Listen("tcp", "127.0.0.1:0")
	require.NoError(t, err)
	t.Cleanup(func() { l.Close() })
	go Proxy(l, d, a)
	return l
}

func TestProxyRoundTrip(t *testing.T) {
	ctx, done := context.WithTimeout(context.Background(), 5*time.Second)
	defer done()
	target := echoServer(t)
	p := proxyServer(t, netdialer{}, allow{"secret"})

	c, err := ProxyDialer{Proxy: p.Addr().String(), Signer: signed("secret"), Dialer: netdialer{}}.DialContext(ctx, "tcp", target.Addr().String())
	require.NoError(t, err)
	defer c.Close()

	_, err = c.Write([]byte("hello"))
	require.NoError(t, err)
	buf := make([]byte, 5)
	c.SetReadDeadline(time.Now().Add(2 * time.Second))
	_, err = io.ReadFull(c, buf)
	require.NoError(t, err)
	require.Equal(t, "hello", string(buf))
}

func TestProxyRejectsBadToken(t *testing.T) {
	ctx, done := context.WithTimeout(context.Background(), 5*time.Second)
	defer done()
	d := refuse{addr: make(chan string, 1)}
	p := proxyServer(t, d, allow{"secret"})

	_, err := ProxyDialer{Proxy: p.Addr().String(), Signer: signed("wrong"), Dialer: netdialer{}}.DialContext(ctx, "tcp", "10.0.0.1:2000")
	require.Error(t, err)

	// an unauthorized request must never reach the dialer.
	select {
	case a := <-d.addr:
		t.Fatalf("unauthorized request dialed %s", a)
	case <-time.After(200 * time.Millisecond):
	}
}

func TestProxyTargetDialFailure(t *testing.T) {
	ctx, done := context.WithTimeout(context.Background(), 5*time.Second)
	defer done()
	d := refuse{addr: make(chan string, 1)}
	p := proxyServer(t, d, allow{"secret"})

	_, err := ProxyDialer{Proxy: p.Addr().String(), Signer: signed("secret"), Dialer: netdialer{}}.DialContext(ctx, "tcp", "10.0.0.1:2000")
	require.Error(t, err)
	require.Equal(t, "10.0.0.1:2000", <-d.addr)
}

// garbage in place of a proxy request closes the connection and the listener keeps serving.
func TestProxyMalformedRequest(t *testing.T) {
	ctx, done := context.WithTimeout(context.Background(), 5*time.Second)
	defer done()
	target := echoServer(t)
	p := proxyServer(t, netdialer{}, allow{"secret"})

	raw, err := net.Dial("tcp", p.Addr().String())
	require.NoError(t, err)
	raw.Write([]byte{1, 2, 3})
	raw.Close()

	c, err := ProxyDialer{Proxy: p.Addr().String(), Signer: signed("secret"), Dialer: netdialer{}}.DialContext(ctx, "tcp", target.Addr().String())
	require.NoError(t, err)
	c.Close()
}
