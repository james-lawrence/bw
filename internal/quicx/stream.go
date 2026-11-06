package quicx

import (
	"context"
	"crypto/tls"
	"errors"
	"net"
	"net/netip"
	"os"
	"sync"
	"time"

	"golang.org/x/net/quic"
)

// closeTimeout bounds how long Close waits for the peer to acknowledge the stream.
const closeTimeout = 5 * time.Second

func udpaddr(a netip.AddrPort) *net.UDPAddr {
	return net.UDPAddrFromAddrPort(a)
}

// stream adapts a quic stream to a net.Conn. it also implements Handshake and
// ConnectionState so the muxer can verify the negotiated protocol the same way
// it does for tls connections.
//
// deadlines apply to read and write calls which start after the deadline is set,
// a call already blocked is not affected.
type stream struct {
	s *quic.Stream
	c *quic.Conn

	m      sync.Mutex
	rdl    time.Time
	wdl    time.Time
	closed sync.Once
	cerr   error
}

func newStream(c *quic.Conn, s *quic.Stream) *stream {
	return &stream{c: c, s: s}
}

func (t *stream) deadlines() (r time.Time, w time.Time) {
	t.m.Lock()
	defer t.m.Unlock()
	return t.rdl, t.wdl
}

func deadlineContext(dl time.Time) (context.Context, context.CancelFunc) {
	if dl.IsZero() {
		return context.Background(), func() {}
	}

	return context.WithDeadline(context.Background(), dl)
}

func translate(err error, ctx context.Context) error {
	if err == nil {
		return nil
	}

	if errors.Is(err, context.DeadlineExceeded) || ctx.Err() == context.DeadlineExceeded {
		return os.ErrDeadlineExceeded
	}

	return err
}

func (t *stream) Read(b []byte) (n int, err error) {
	rdl, _ := t.deadlines()
	ctx, done := deadlineContext(rdl)
	defer done()
	t.s.SetReadContext(ctx)
	n, err = t.s.Read(b)
	return n, translate(err, ctx)
}

func (t *stream) Write(b []byte) (n int, err error) {
	_, wdl := t.deadlines()
	ctx, done := deadlineContext(wdl)
	defer done()
	t.s.SetWriteContext(ctx)
	if n, err = t.s.Write(b); err != nil {
		return n, translate(err, ctx)
	}

	// a quic stream is not announced to the peer until data is flushed, flush on every write
	// to match the semantics of a tcp connection.
	return n, translate(t.s.Flush(), ctx)
}

func (t *stream) Close() error {
	t.closed.Do(func() {
		ctx, done := context.WithTimeout(context.Background(), closeTimeout)
		defer done()
		t.s.SetWriteContext(ctx)
		t.cerr = t.s.Close()
	})

	return t.cerr
}

func (t *stream) LocalAddr() net.Addr {
	return udpaddr(t.c.LocalAddr())
}

func (t *stream) RemoteAddr() net.Addr {
	return udpaddr(t.c.RemoteAddr())
}

func (t *stream) SetDeadline(dl time.Time) error {
	t.m.Lock()
	defer t.m.Unlock()
	t.rdl, t.wdl = dl, dl
	return nil
}

func (t *stream) SetReadDeadline(dl time.Time) error {
	t.m.Lock()
	defer t.m.Unlock()
	t.rdl = dl
	return nil
}

func (t *stream) SetWriteDeadline(dl time.Time) error {
	t.m.Lock()
	defer t.m.Unlock()
	t.wdl = dl
	return nil
}

// Handshake is a noop, the quic connection handshake completes before streams exist.
func (t *stream) Handshake() error {
	return nil
}

// ConnectionState returns the tls state of the underlying quic connection.
func (t *stream) ConnectionState() tls.ConnectionState {
	return t.c.ConnectionState()
}
