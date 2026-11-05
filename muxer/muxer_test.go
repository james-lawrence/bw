package muxer_test

import (
	"context"
	"io"
	"log"
	"net"
	"os"
	"sync/atomic"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	"google.golang.org/protobuf/proto"

	"github.com/james-lawrence/bw/internal/testingx"
	. "github.com/james-lawrence/bw/muxer"
)

func TestMain(m *testing.M) {
	log.SetOutput(io.Discard)
	code := m.Run()
	testingx.Cleanup()
	os.Exit(code)
}

func counter(l net.Listener, c *int64) {
	for {
		_, err := l.Accept()
		if err != nil {
			return
		}
		atomic.AddInt64(c, 1)
	}
}

func TestParseURI(t *testing.T) {
	t.Run("should extract the procol name and address", func(t *testing.T) {
		proto, host, err := ParseURI("bw.agent://QmNMbFdN9s7R1PBr9VqZqVrsZzunwh61hD1vTR96AmohGX")
		require.NoError(t, err)
		require.Equal(t, "bw.agent", string(proto))
		require.Equal(t, "QmNMbFdN9s7R1PBr9VqZqVrsZzunwh61hD1vTR96AmohGX", host)
	})
}

func TestRebind(t *testing.T) {
	t.Run("should be able to rebind a protocol successfully", func(t *testing.T) {
		c1 := int64(0)
		c2 := int64(0)

		m := New()
		l, err := net.Listen("tcp", ":0")
		require.NoError(t, err)
		defer l.Close()
		l1, err := m.Rebind("proto", l.Addr())
		require.NoError(t, err)
		defer l1.Close()
		l2, err := m.Rebind("proto", l.Addr())
		require.NoError(t, err)
		defer l2.Close()
		go func() {
			go counter(l1, &c1)
			go counter(l2, &c2)
			Listen(context.Background(), m, l)
		}()

		d1 := NewDialer("proto", &net.Dialer{})

		for i := 0; i < 10; i++ {
			conn, err := d1.DialContext(context.Background(), "tcp", l.Addr().String())
			require.NoError(t, err)
			require.NoError(t, conn.Close())
			require.Equal(t, int64(0), atomic.LoadInt64(&c1))
			require.Eventually(t, func() bool { return atomic.LoadInt64(&c2) == int64(i+1) }, time.Second, 10*time.Millisecond)
		}
	})
}

func TestDialAndAccept(t *testing.T) {
	t.Run("should be able to multiplex", func(t *testing.T) {
		c1 := int64(0)
		c2 := int64(0)

		m := New()
		l, err := net.Listen("tcp", ":0")
		require.NoError(t, err)
		defer l.Close()
		l1, err := m.Bind("proto1", l.Addr())
		require.NoError(t, err)
		defer l1.Close()
		l2, err := m.Bind("proto2", l.Addr())
		require.NoError(t, err)
		defer l2.Close()
		go func() {
			go counter(l1, &c1)
			go counter(l2, &c2)
			Listen(context.Background(), m, l)
		}()

		d1 := NewDialer("proto1", &net.Dialer{})

		conn, err := d1.DialContext(context.Background(), "tcp", l.Addr().String())
		require.NoError(t, err)
		require.NoError(t, conn.Close())
		require.Eventually(t, func() bool { return atomic.LoadInt64(&c1) == 1 }, time.Second, 10*time.Millisecond)
		require.Equal(t, int64(0), atomic.LoadInt64(&c2))

		d2 := NewDialer("proto2", &net.Dialer{})

		conn, err = d2.DialContext(context.Background(), "tcp", l.Addr().String())
		require.NoError(t, err)
		require.NoError(t, conn.Close())
		require.Eventually(t, func() bool { return atomic.LoadInt64(&c2) == 1 }, time.Second, 10*time.Millisecond)
		require.Equal(t, int64(1), atomic.LoadInt64(&c1))
	})
}

func TestAccepted(t *testing.T) {
	// should encode to a fixed size (22)
	for _, c := range []struct {
		name  string
		proto string
		code  AcceptedError
	}{
		{"example 1 - default protocol", "", Accepted_None},
		{"example 2 - long name", "45b3058c-9ec4-41cc-bd4c-c74ca2abdea2", Accepted_None},
		{"example 3 - client error", "45b3058c-9ec4-41cc-bd4c-c74ca2abdea2", Accepted_ClientError},
		{"example 4 - server error", "45b3058c-9ec4-41cc-bd4c-c74ca2abdea2", Accepted_ServerError},
	} {
		t.Run(c.name, func(t *testing.T) {
			protocol := Proto(c.proto)
			encoded, err := proto.Marshal(&Accepted{
				Version:  1,
				Code:     c.code,
				Protocol: protocol[:],
			})
			require.NoError(t, err)
			require.Len(t, encoded, 22)
		})
	}
}

func TestRequested(t *testing.T) {
	// should encode to a fixed size (20)
	for _, c := range []struct {
		name  string
		proto string
	}{
		{"example 1 - default protocol", ""},
		{"example 2 - long name", "45b3058c-9ec4-41cc-bd4c-c74ca2abdea2"},
	} {
		t.Run(c.name, func(t *testing.T) {
			protocol := Proto(c.proto)
			encoded, err := proto.Marshal(&Requested{
				Version:  1,
				Protocol: protocol[:],
			})
			require.NoError(t, err)
			require.Len(t, encoded, 20)
		})
	}
}
