package quicx

import (
	"net"
	"os"
	"sync"
	"time"
)

// quicBit is the fixed bit set in the first byte of every quic packet (long and short headers).
// memberlist packets lead with a message type or encryption version which is always below it.
const quicBit = 0x40

// Demux shares a single udp socket between quic and a legacy packet protocol (swim).
// inbound packets are routed by their first byte, outbound packets go straight to the socket.
// closing either side closes the socket.
func Demux(pc net.PacketConn) (quic net.PacketConn, legacy net.PacketConn) {
	d := &demux{pc: pc, done: make(chan struct{})}
	d.quic = newSide(d)
	d.legacy = newSide(d)
	go d.run()
	return d.quic, d.legacy
}

type inbound struct {
	b    []byte
	addr net.Addr
}

type demux struct {
	pc     net.PacketConn
	quic   *side
	legacy *side
	done   chan struct{}
	once   sync.Once
}

func (t *demux) run() {
	buf := make([]byte, 65536)
	for {
		n, addr, err := t.pc.ReadFrom(buf)
		if err != nil {
			t.close()
			return
		}

		if n == 0 {
			continue
		}

		dst := t.legacy
		if buf[0]&quicBit != 0 {
			dst = t.quic
		}

		select {
		case dst.in <- inbound{b: append([]byte(nil), buf[:n]...), addr: addr}:
		default:
			// receiver is behind, drop like the kernel would.
		}
	}
}

func (t *demux) close() error {
	var err error
	t.once.Do(func() {
		close(t.done)
		err = t.pc.Close()
	})
	return err
}

type side struct {
	d  *demux
	in chan inbound

	m  sync.Mutex
	dl time.Time
}

func newSide(d *demux) *side {
	return &side{d: d, in: make(chan inbound, 512)}
}

func (t *side) ReadFrom(b []byte) (int, net.Addr, error) {
	t.m.Lock()
	dl := t.dl
	t.m.Unlock()

	var expired <-chan time.Time
	if !dl.IsZero() {
		timer := time.NewTimer(time.Until(dl))
		defer timer.Stop()
		expired = timer.C
	}

	select {
	case p := <-t.in:
		return copy(b, p.b), p.addr, nil
	case <-expired:
		return 0, nil, os.ErrDeadlineExceeded
	case <-t.d.done:
		return 0, nil, net.ErrClosed
	}
}

func (t *side) WriteTo(b []byte, addr net.Addr) (int, error) {
	return t.d.pc.WriteTo(b, addr)
}

func (t *side) Close() error                        { return t.d.close() }
func (t *side) LocalAddr() net.Addr                 { return t.d.pc.LocalAddr() }
func (t *side) SetWriteDeadline(dl time.Time) error { return t.d.pc.SetWriteDeadline(dl) }

func (t *side) SetDeadline(dl time.Time) error {
	return t.SetReadDeadline(dl)
}

func (t *side) SetReadDeadline(dl time.Time) error {
	t.m.Lock()
	defer t.m.Unlock()
	t.dl = dl
	return nil
}
