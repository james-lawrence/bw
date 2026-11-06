// Package quicx carries bw's connections over a single quic socket. it is the only package
// which imports the quic implementation.
//
// a quic connection between two nodes is shared, every net.Conn is a stream of it. the
// first bytes of each stream are the muxer header, exactly as for a tls connection.
// datagram style packets (swim) are sent as short lived unidirectional streams.
package quicx

import (
	"context"
	"crypto/tls"
	"io"
	"net"
	"sync"
	"time"

	"github.com/pkg/errors"
	"golang.org/x/net/quic"
)

const (
	// ALPN is the protocol negotiated by the quic handshake, it matches the tls listener.
	ALPN = "bw.mux"
	// MaxPacket limits the size of a packet received over a unidirectional stream.
	MaxPacket = 64 * 1024

	handshakeTimeout = 3 * time.Second
	// unavailableFor is how long an address which failed to complete a quic handshake is
	// skipped, peers without quic (older versions, udp blocked) cost one timeout, not one per dial.
	unavailableFor = 30 * time.Second
	idleTimeout    = 30 * time.Second
	keepAlive      = 10 * time.Second
)

var (
	errClosed      = errors.New("quicx: endpoint closed")
	errUnavailable = errors.New("quicx: address recently failed to negotiate quic")
)

// Packet is a single message received over a unidirectional stream.
type Packet struct {
	Data []byte
	Addr net.Addr
}

func harden(c *tls.Config) *tls.Config {
	c = c.Clone()
	c.MinVersion = tls.VersionTLS13
	c.NextProtos = []string{ALPN}
	return c
}

// Endpoint owns a quic endpoint on a packet connection. it accepts connections, dials
// pooled connections and implements net.Listener where Accept returns accepted streams.
type Endpoint struct {
	e      *quic.Endpoint
	pc     net.PacketConn
	client *quic.Config

	ctx     context.Context
	cancel  context.CancelFunc
	streams chan net.Conn
	packets chan Packet

	m      sync.Mutex
	conns  map[string]*pooled
	failed map[string]time.Time
}

type pooled struct {
	ready chan struct{}
	c     *quic.Conn
	err   error
}

// New starts a quic endpoint on the packet connection. server is the tls configuration used for
// accepted connections, client is used for dialed connections. the endpoint takes ownership of pc.
func New(pc net.PacketConn, server *tls.Config, client *tls.Config) (*Endpoint, error) {
	cfg := func(t *tls.Config) *quic.Config {
		return &quic.Config{
			TLSConfig:        harden(t),
			HandshakeTimeout: handshakeTimeout,
			MaxIdleTimeout:   idleTimeout,
			KeepAlivePeriod:  keepAlive,
		}
	}

	e, err := quic.NewEndpoint(pc, cfg(server))
	if err != nil {
		return nil, errors.Wrap(err, "unable to start quic endpoint")
	}

	ctx, cancel := context.WithCancel(context.Background())
	t := &Endpoint{
		e:       e,
		pc:      pc,
		client:  cfg(client),
		ctx:     ctx,
		cancel:  cancel,
		streams: make(chan net.Conn),
		packets: make(chan Packet, 128),
		conns:   make(map[string]*pooled),
		failed:  make(map[string]time.Time),
	}

	go t.acceptConns()

	return t, nil
}

func (t *Endpoint) acceptConns() {
	for {
		c, err := t.e.Accept(t.ctx)
		if err != nil {
			return
		}

		go t.acceptStreams(c)
	}
}

func (t *Endpoint) acceptStreams(c *quic.Conn) {
	defer c.Abort(nil)

	for {
		s, err := c.AcceptStream(t.ctx)
		if err != nil {
			return
		}

		if s.IsReadOnly() {
			go t.receivePacket(c, s)
			continue
		}

		select {
		case t.streams <- newStream(c, s):
		case <-t.ctx.Done():
			return
		}
	}
}

func (t *Endpoint) receivePacket(c *quic.Conn, s *quic.Stream) {
	ctx, done := context.WithTimeout(t.ctx, handshakeTimeout)
	defer done()
	s.SetReadContext(ctx)
	defer s.CloseRead()

	b, err := io.ReadAll(io.LimitReader(s, MaxPacket))
	if err != nil || len(b) == 0 {
		return
	}

	select {
	case t.packets <- Packet{Data: b, Addr: udpaddr(c.RemoteAddr())}:
	default:
		// the receiver is behind, drop like udp would.
	}
}

// Accept the next inbound stream.
func (t *Endpoint) Accept() (net.Conn, error) {
	select {
	case c := <-t.streams:
		return c, nil
	case <-t.ctx.Done():
		return nil, errClosed
	}
}

// Addr the local address of the endpoint.
func (t *Endpoint) Addr() net.Addr {
	return t.pc.LocalAddr()
}

// Close the endpoint and all of its connections.
func (t *Endpoint) Close() error {
	t.cancel()
	ctx, done := context.WithTimeout(context.Background(), time.Second)
	defer done()
	return t.e.Close(ctx)
}

// conn returns the pooled connection for the address, dialing when there is none.
func (t *Endpoint) conn(ctx context.Context, address string) (*quic.Conn, error) {
	t.m.Lock()
	p, ok := t.conns[address]
	if !ok {
		if until, bad := t.failed[address]; bad {
			if time.Now().Before(until) {
				t.m.Unlock()
				return nil, errUnavailable
			}
			delete(t.failed, address)
		}

		p = &pooled{ready: make(chan struct{})}
		t.conns[address] = p
		go func() {
			defer close(p.ready)
			p.c, p.err = t.e.Dial(t.ctx, "udp", address, t.client)
			if p.err != nil {
				t.m.Lock()
				delete(t.conns, address)
				t.failed[address] = time.Now().Add(unavailableFor)
				t.m.Unlock()
				return
			}

			// forget the connection once it terminates so the next dial replaces it.
			go func(c *quic.Conn) {
				_ = c.Wait(t.ctx)
				t.evict(address, c)
			}(p.c)
		}()
	}
	t.m.Unlock()

	select {
	case <-p.ready:
	case <-ctx.Done():
		return nil, ctx.Err()
	}

	return p.c, p.err
}

func (t *Endpoint) evict(address string, c *quic.Conn) {
	t.m.Lock()
	defer t.m.Unlock()
	if p, ok := t.conns[address]; ok && p.c == c {
		delete(t.conns, address)
	}
	c.Abort(nil)
}

// DialContext opens a new stream to the address on a pooled connection. the network is ignored,
// quic is always udp. a pooled connection which has gone stale is replaced once.
func (t *Endpoint) DialContext(ctx context.Context, network string, address string) (net.Conn, error) {
	var lasterr error
	for attempt := 0; attempt < 2; attempt++ {
		c, err := t.conn(ctx, address)
		if err != nil {
			return nil, errors.Wrapf(err, "quicx: dial failed: %s", address)
		}

		s, err := c.NewStream(ctx)
		if err != nil {
			t.evict(address, c)
			lasterr = err
			continue
		}

		return newStream(c, s), nil
	}

	return nil, errors.Wrapf(lasterr, "quicx: stream open failed: %s", address)
}

// SendPacket delivers a single message as a unidirectional stream.
func (t *Endpoint) SendPacket(ctx context.Context, address string, b []byte) (err error) {
	var lasterr error
	for attempt := 0; attempt < 2; attempt++ {
		c, err := t.conn(ctx, address)
		if err != nil {
			return errors.Wrapf(err, "quicx: dial failed: %s", address)
		}

		s, err := c.NewSendOnlyStream(ctx)
		if err != nil {
			t.evict(address, c)
			lasterr = err
			continue
		}

		s.SetWriteContext(ctx)
		if _, err = s.Write(b); err != nil {
			s.Reset(0)
			return err
		}

		// close the write side without waiting for the acknowledgement.
		return s.CloseWrite()
	}

	return errors.Wrapf(lasterr, "quicx: stream open failed: %s", address)
}

// ReceivePacket blocks until a packet arrives.
func (t *Endpoint) ReceivePacket(ctx context.Context) (Packet, error) {
	select {
	case p := <-t.packets:
		return p, nil
	case <-ctx.Done():
		return Packet{}, ctx.Err()
	case <-t.ctx.Done():
		return Packet{}, errClosed
	}
}

// Established reports whether a live quic connection to the address is pooled.
func (t *Endpoint) Established(address string) bool {
	t.m.Lock()
	p, ok := t.conns[address]
	t.m.Unlock()
	if !ok {
		return false
	}

	select {
	case <-p.ready:
		return p.err == nil
	default:
		return false
	}
}

// Warm starts connecting to the address in the background, it never blocks.
func (t *Endpoint) Warm(address string) {
	go func() {
		ctx, done := context.WithTimeout(t.ctx, handshakeTimeout)
		defer done()
		_, _ = t.conn(ctx, address)
	}()
}

type dialer interface {
	DialContext(ctx context.Context, network string, address string) (net.Conn, error)
}

// Fallback prefers quic streams and falls back to the dialer when the address does not
// support quic. addresses which fail are skipped for a while, see unavailableFor.
func (t *Endpoint) Fallback(d dialer) dialer {
	return fallback{e: t, d: d}
}

type fallback struct {
	e *Endpoint
	d dialer
}

func (t fallback) DialContext(ctx context.Context, network string, address string) (net.Conn, error) {
	qctx, done := context.WithTimeout(ctx, handshakeTimeout)
	defer done()

	if c, err := t.e.DialContext(qctx, network, address); err == nil {
		return c, nil
	}

	if ctx.Err() != nil {
		return nil, ctx.Err()
	}

	return t.d.DialContext(ctx, network, address)
}
