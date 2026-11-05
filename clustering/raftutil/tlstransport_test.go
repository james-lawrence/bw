package raftutil

import (
	"crypto/tls"
	"net"
	"os"
	"testing"
	"time"

	"github.com/hashicorp/raft"
	"github.com/stretchr/testify/require"
)

func TestTLSTransport(t *testing.T) {
	t.Run("should gracefully close and reopen a net listenr", func(t *testing.T) {
		c := &tls.Config{
			InsecureSkipVerify: true,
		}

		l, err := net.Listen("tcp", "127.1.1.2:9999")
		require.NoError(t, err)
		sl := NewStreamTransport(l, NewTLSStreamDialer(c))

		nt := raft.NewNetworkTransport(sl, 5, 10*time.Second, os.Stderr)
		require.NoError(t, autocloseTransport(nt))

		l, err = net.Listen("tcp", "127.1.1.2:9999")
		require.NoError(t, err)

		sl = NewStreamTransport(l, NewTLSStreamDialer(c))
		nt = raft.NewNetworkTransport(sl, 5, 10*time.Second, os.Stderr)
		require.NoError(t, autocloseTransport(nt))
	})
}
