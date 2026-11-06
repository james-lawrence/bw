package proxy_test

import (
	"net"
	"testing"
	"time"

	"github.com/james-lawrence/bw/proxy"
	"github.com/stretchr/testify/require"
)

func TestProxyShouldReturnAndCloseWhenOneSideCloses(t *testing.T) {
	c1, c1peer := net.Pipe()
	c2, c2peer := net.Pipe()

	done := make(chan error, 1)
	go func() { done <- proxy.Proxy(t.Context(), c1, c2, nil) }()

	go func() { c1peer.Write([]byte("ping")) }()
	buf := make([]byte, 4)
	_, err := c2peer.Read(buf)
	require.NoError(t, err)
	require.Equal(t, "ping", string(buf))

	require.NoError(t, c1peer.Close())

	select {
	case <-done:
	case <-time.After(5 * time.Second):
		t.Fatal("proxy did not return")
	}

	_, err = c2peer.Read(buf)
	require.Error(t, err, "dst should have been closed")
}
