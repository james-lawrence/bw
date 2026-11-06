package quicx_test

import (
	"context"
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/tls"
	"crypto/x509"
	"crypto/x509/pkix"
	"io"
	"math/big"
	"net"
	"os"
	"sync"
	"testing"
	"time"

	"github.com/james-lawrence/bw/internal/quicx"
	"github.com/james-lawrence/bw/muxer"
	"github.com/stretchr/testify/require"
)

var once sync.Once
var shared struct{ server, client *tls.Config }

func tlsconfigs(t testing.TB) (server *tls.Config, client *tls.Config) {
	once.Do(func() { shared.server, shared.client = gen(t) })
	return shared.server, shared.client
}

func gen(t testing.TB) (server *tls.Config, client *tls.Config) {
	k, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	require.NoError(t, err)
	tmpl := &x509.Certificate{SerialNumber: big.NewInt(1), Subject: pkix.Name{CommonName: "bw"}, NotBefore: time.Now().Add(-time.Hour), NotAfter: time.Now().Add(time.Hour), DNSNames: []string{"bw"}, IsCA: true, BasicConstraintsValid: true, KeyUsage: x509.KeyUsageCertSign | x509.KeyUsageDigitalSignature}
	der, err := x509.CreateCertificate(rand.Reader, tmpl, tmpl, &k.PublicKey, k)
	require.NoError(t, err)
	crt, err := x509.ParseCertificate(der)
	require.NoError(t, err)
	pool := x509.NewCertPool()
	pool.AddCert(crt)

	return &tls.Config{Certificates: []tls.Certificate{{Certificate: [][]byte{der}, PrivateKey: k}}},
		&tls.Config{RootCAs: pool, ServerName: "bw"}
}

func endpoint(t testing.TB) *quicx.Endpoint {
	s, c := tlsconfigs(t)
	pc, err := net.ListenPacket("udp", "127.0.0.1:0")
	require.NoError(t, err)
	e, err := quicx.New(pc, s, c)
	require.NoError(t, err)
	t.Cleanup(func() { e.Close() })
	return e
}

func TestStreamRoundTrip(t *testing.T) {
	ctx, done := context.WithTimeout(context.Background(), 10*time.Second)
	defer done()
	a, b := endpoint(t), endpoint(t)

	go func() {
		c, err := b.Accept()
		if err != nil {
			return
		}
		defer c.Close()
		io.Copy(c, c)
	}()

	c, err := a.DialContext(ctx, "udp", b.Addr().String())
	require.NoError(t, err)
	defer c.Close()
	_, err = c.Write([]byte("hello"))
	require.NoError(t, err)
	buf := make([]byte, 5)
	_, err = io.ReadFull(c, buf)
	require.NoError(t, err)
	require.Equal(t, "hello", string(buf))
}

func TestReadDeadline(t *testing.T) {
	ctx, done := context.WithTimeout(context.Background(), 10*time.Second)
	defer done()
	a, b := endpoint(t), endpoint(t)
	go func() {
		if c, err := b.Accept(); err == nil {
			io.Copy(io.Discard, c)
		}
	}()

	c, err := a.DialContext(ctx, "udp", b.Addr().String())
	require.NoError(t, err)
	defer c.Close()
	_, err = c.Write([]byte("x"))
	require.NoError(t, err)
	require.NoError(t, c.SetReadDeadline(time.Now().Add(100*time.Millisecond)))
	_, err = c.Read(make([]byte, 1))
	require.ErrorIs(t, err, os.ErrDeadlineExceeded)
}

func TestPacket(t *testing.T) {
	ctx, done := context.WithTimeout(context.Background(), 10*time.Second)
	defer done()
	a, b := endpoint(t), endpoint(t)
	require.NoError(t, a.SendPacket(ctx, b.Addr().String(), []byte("swim")))
	p, err := b.ReceivePacket(ctx)
	require.NoError(t, err)
	require.Equal(t, "swim", string(p.Data))
}

// the muxer works unchanged over quic streams.
func TestMuxerOverQuic(t *testing.T) {
	ctx, done := context.WithTimeout(context.Background(), 10*time.Second)
	defer done()
	a, b := endpoint(t), endpoint(t)

	m := muxer.New()
	l, err := m.Bind("bw.test", b.Addr())
	require.NoError(t, err)
	go muxer.Listen(ctx, m, b)

	go func() {
		c, err := l.Accept()
		if err != nil {
			return
		}
		defer c.Close()
		io.Copy(c, c)
	}()

	c, err := muxer.NewDialer("bw.test", a).DialContext(ctx, "udp", b.Addr().String())
	require.NoError(t, err)
	defer c.Close()
	_, err = c.Write([]byte("ping"))
	require.NoError(t, err)
	buf := make([]byte, 4)
	_, err = io.ReadFull(c, buf)
	require.NoError(t, err)
	require.Equal(t, "ping", string(buf))
}

// a peer which restarted is replaced transparently on the next dial.
func TestStaleConnectionRedial(t *testing.T) {
	ctx, done := context.WithTimeout(context.Background(), 20*time.Second)
	defer done()
	s, c := tlsconfigs(t)
	pc, err := net.ListenPacket("udp", "127.0.0.1:0")
	require.NoError(t, err)
	addr := pc.LocalAddr().String()
	b, err := quicx.New(pc, s, c)
	require.NoError(t, err)
	a := endpoint(t)

	go func() {
		if c, err := b.Accept(); err == nil {
			io.Copy(io.Discard, c)
		}
	}()
	c1, err := a.DialContext(ctx, "udp", addr)
	require.NoError(t, err)
	_, err = c1.Write([]byte("x"))
	require.NoError(t, err)
	b.Close()

	pc2, err := net.ListenPacket("udp", addr)
	require.NoError(t, err)
	b2, err := quicx.New(pc2, s, c)
	require.NoError(t, err)
	defer b2.Close()
	go func() {
		if c, err := b2.Accept(); err == nil {
			io.Copy(io.Discard, c)
		}
	}()

	var c2 net.Conn
	require.Eventually(t, func() bool {
		dctx, dcancel := context.WithTimeout(ctx, 5*time.Second)
		defer dcancel()
		c2, err = a.DialContext(dctx, "udp", addr)
		if err != nil {
			return false
		}
		_, err = c2.Write([]byte("y"))
		return err == nil
	}, 15*time.Second, 200*time.Millisecond)
}

type failing struct{ called int }

func (t *failing) DialContext(ctx context.Context, network, address string) (net.Conn, error) {
	t.called++
	return nil, io.EOF
}

// legacy packets and quic share one socket.
func TestDemux(t *testing.T) {
	ctx, done := context.WithTimeout(context.Background(), 10*time.Second)
	defer done()
	s, c := tlsconfigs(t)
	pc, err := net.ListenPacket("udp", "127.0.0.1:0")
	require.NoError(t, err)
	qpc, legacy := quicx.Demux(pc)
	b, err := quicx.New(qpc, s, c)
	require.NoError(t, err)
	defer b.Close()
	a := endpoint(t)

	raw, err := net.Dial("udp", pc.LocalAddr().String())
	require.NoError(t, err)
	_, err = raw.Write([]byte{0x01, 'h', 'i'})
	require.NoError(t, err)
	buf := make([]byte, 16)
	legacy.SetReadDeadline(time.Now().Add(2 * time.Second))
	n, _, err := legacy.ReadFrom(buf)
	require.NoError(t, err)
	require.Equal(t, []byte{0x01, 'h', 'i'}, buf[:n])

	require.NoError(t, a.SendPacket(ctx, pc.LocalAddr().String(), []byte("swim")))
	p, err := b.ReceivePacket(ctx)
	require.NoError(t, err)
	require.Equal(t, "swim", string(p.Data))
}

// an address without quic falls back once and is then skipped.
func TestFallback(t *testing.T) {
	ctx, done := context.WithTimeout(context.Background(), 20*time.Second)
	defer done()
	a := endpoint(t)
	// a udp socket nothing answers on.
	dead, err := net.ListenPacket("udp", "127.0.0.1:0")
	require.NoError(t, err)
	defer dead.Close()

	f := &failing{}
	d := a.Fallback(f)
	_, err = d.DialContext(ctx, "tcp", dead.LocalAddr().String())
	require.ErrorIs(t, err, io.EOF)
	start := time.Now()
	_, err = d.DialContext(ctx, "tcp", dead.LocalAddr().String())
	require.ErrorIs(t, err, io.EOF)
	require.Less(t, time.Since(start), time.Second)
	require.Equal(t, 2, f.called)
}
