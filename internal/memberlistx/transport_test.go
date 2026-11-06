package memberlistx_test

import (
	"context"
	"net"
	"sync"
	"testing"
	"time"

	"github.com/james-lawrence/bw/internal/memberlistx"
	"github.com/james-lawrence/bw/internal/quicx"
	"github.com/stretchr/testify/require"
)

type nodialer struct{}

func (nodialer) DialContext(ctx context.Context, network, address string) (net.Conn, error) {
	return (&net.Dialer{}).DialContext(ctx, network, address)
}

func udp(t *testing.T) net.PacketConn {
	pc, err := net.ListenPacket("udp", "127.0.0.1:0")
	require.NoError(t, err)
	return pc
}

func tcp(t *testing.T) net.Listener {
	l, err := net.Listen("tcp", "127.0.0.1:0")
	require.NoError(t, err)
	return l
}

func transport(t *testing.T, opts ...memberlistx.SWIMTransportOption) (*memberlistx.SWIMTransport, net.PacketConn, net.Listener) {
	pc, l := udp(t), tcp(t)
	opts = append([]memberlistx.SWIMTransportOption{memberlistx.SWIMStreams(l), memberlistx.SWIMPackets(pc)}, opts...)
	tr, err := memberlistx.NewSWIMTransport(nodialer{}, opts...)
	require.NoError(t, err)
	t.Cleanup(func() { tr.Shutdown() })
	return tr, pc, l
}

func TestConstructionRequiresTransports(t *testing.T) {
	_, err := memberlistx.NewSWIMTransport(nil, memberlistx.SWIMStreams(tcp(t)), memberlistx.SWIMPackets(udp(t)))
	require.Error(t, err, "dialer required")
	_, err = memberlistx.NewSWIMTransport(nodialer{}, memberlistx.SWIMPackets(udp(t)))
	require.Error(t, err, "stream transport required")
	_, err = memberlistx.NewSWIMTransport(nodialer{}, memberlistx.SWIMStreams(tcp(t)))
	require.Error(t, err, "packet transport required")
}

func TestPacketsOverUDP(t *testing.T) {
	a, _, _ := transport(t)
	b, bpc, _ := transport(t)

	_, err := a.WriteTo([]byte("ping"), bpc.LocalAddr().String())
	require.NoError(t, err)

	select {
	case p := <-b.PacketCh():
		require.Equal(t, "ping", string(p.Buf))
		require.NotZero(t, p.Timestamp)
	case <-time.After(2 * time.Second):
		t.Fatal("packet not received")
	}
}

func TestStreams(t *testing.T) {
	a, _, _ := transport(t)
	b, _, bl := transport(t)

	go func() {
		c, err := a.DialTimeout(bl.Addr().String(), time.Second)
		if err == nil {
			c.Write([]byte("x"))
			c.Close()
		}
	}()

	select {
	case c := <-b.StreamCh():
		buf := make([]byte, 1)
		_, err := c.Read(buf)
		require.NoError(t, err)
		require.Equal(t, "x", string(buf))
		c.Close()
	case <-time.After(2 * time.Second):
		t.Fatal("stream not accepted")
	}
}

// every packet connection is serviced, not just the first.
func TestMultiplePacketConns(t *testing.T) {
	p1, p2, l := udp(t), udp(t), tcp(t)
	tr, err := memberlistx.NewSWIMTransport(nodialer{}, memberlistx.SWIMStreams(l, tcp(t)), memberlistx.SWIMPackets(p1, p2))
	require.NoError(t, err)
	defer tr.Shutdown()

	w := udp(t)
	defer w.Close()
	w.WriteTo([]byte("two"), p2.LocalAddr())

	select {
	case p := <-tr.PacketCh():
		require.Equal(t, "two", string(p.Buf))
	case <-time.After(2 * time.Second):
		t.Fatal("second packet conn is not being read")
	}
}

func TestFinalAdvertiseAddr(t *testing.T) {
	tr, pc, _ := transport(t)

	ip, port, err := tr.FinalAdvertiseAddr("10.1.2.3", 4000)
	require.NoError(t, err)
	require.Equal(t, "10.1.2.3", ip.String())
	require.Equal(t, 4000, port)

	_, _, err = tr.FinalAdvertiseAddr("not-an-ip", 1)
	require.Error(t, err)

	// bound to loopback, advertise that address and the bound port.
	ip, port, err = tr.FinalAdvertiseAddr("", 0)
	require.NoError(t, err)
	require.Equal(t, "127.0.0.1", ip.String())
	require.Equal(t, pc.LocalAddr().(*net.UDPAddr).Port, port)
	require.Equal(t, port, tr.GetAutoBindPort())
}

func TestShutdownStopsTransport(t *testing.T) {
	tr, _, l := transport(t)
	done := make(chan error, 1)
	go func() { done <- tr.Shutdown() }()
	select {
	case err := <-done:
		require.NoError(t, err)
	case <-time.After(2 * time.Second):
		t.Fatal("shutdown blocked")
	}
	_, err := l.Accept()
	require.Error(t, err)
}

type exchange struct {
	m           sync.Mutex
	established bool
	warmed      []string
	sent        [][]byte
	in          chan quicx.Packet
}

func (t *exchange) Established(string) bool { t.m.Lock(); defer t.m.Unlock(); return t.established }
func (t *exchange) Warm(a string)           { t.m.Lock(); defer t.m.Unlock(); t.warmed = append(t.warmed, a) }
func (t *exchange) SendPacket(ctx context.Context, a string, b []byte) error {
	t.m.Lock()
	defer t.m.Unlock()
	t.sent = append(t.sent, b)
	return nil
}
func (t *exchange) ReceivePacket(ctx context.Context) (quicx.Packet, error) {
	select {
	case p := <-t.in:
		return p, nil
	case <-ctx.Done():
		return quicx.Packet{}, ctx.Err()
	}
}

func TestExchangeUsedOnlyWhenEstablished(t *testing.T) {
	x := &exchange{in: make(chan quicx.Packet)}
	a, _, _ := transport(t, memberlistx.SWIMExchange(x))
	b, bpc, _ := transport(t)
	addr := bpc.LocalAddr().String()

	// not established: legacy udp is used and a connection attempt starts.
	_, err := a.WriteTo([]byte("one"), addr)
	require.NoError(t, err)
	select {
	case p := <-b.PacketCh():
		require.Equal(t, "one", string(p.Buf))
	case <-time.After(2 * time.Second):
		t.Fatal("udp fallback packet not received")
	}
	require.Equal(t, []string{addr}, x.warmed)
	require.Empty(t, x.sent)

	// established: the packet bypasses udp.
	x.m.Lock()
	x.established = true
	x.m.Unlock()
	_, err = a.WriteTo([]byte("two"), addr)
	require.NoError(t, err)
	require.Equal(t, [][]byte{[]byte("two")}, x.sent)
	select {
	case p := <-b.PacketCh():
		t.Fatalf("packet %q should not have used udp", p.Buf)
	case <-time.After(200 * time.Millisecond):
	}
}

func TestExchangePacketsReachPacketCh(t *testing.T) {
	x := &exchange{in: make(chan quicx.Packet, 1)}
	a, _, _ := transport(t, memberlistx.SWIMExchange(x))
	from := &net.UDPAddr{IP: net.IPv4(10, 0, 0, 9), Port: 2000}
	x.in <- quicx.Packet{Data: []byte("secure"), Addr: from}

	select {
	case p := <-a.PacketCh():
		require.Equal(t, "secure", string(p.Buf))
		require.Equal(t, from.String(), p.From.String())
	case <-time.After(2 * time.Second):
		t.Fatal("exchange packet not delivered")
	}
}

// more stream listeners than packet connections must not index past the packet connections.
func TestMoreStreamsThanPackets(t *testing.T) {
	tr, err := memberlistx.NewSWIMTransport(nodialer{}, memberlistx.SWIMStreams(tcp(t), tcp(t)), memberlistx.SWIMPackets(udp(t)))
	require.NoError(t, err)
	require.NoError(t, tr.Shutdown())
}
