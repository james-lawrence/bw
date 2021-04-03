package memberlistx

import (
	"context"
	"fmt"
	"io/ioutil"
	"log"
	"net"
	"sync"
	"sync/atomic"
	"time"

	"github.com/hashicorp/go-sockaddr"
	"github.com/hashicorp/memberlist"
	"github.com/pkg/errors"
	"golang.org/x/time/rate"
)

const (
	// udpPacketBufSize is used to buffer incoming packets during read
	// operations.
	udpPacketBufSize = 65536

	// udpRecvBufSize is a large buffer size that we attempt to set UDP
	// sockets to in order to handle a large volume of messages.
	udpRecvBufSize = 2 * 1024 * 1024
)

// NetTransportConfig is used to configure a net transport.
type NetTransportConfig struct {
	// BindAddrs is a list of addresses to bind to for both TCP and UDP
	// communications.
	BindAddrs []string

	// BindPort is the port to listen on, for each address above.
	BindPort int
}

// SWIMTransportOption ...
type SWIMTransportOption func(*SWIMTransport)

// SWIMStreams relieable transports
func SWIMStreams(streams ...net.Listener) SWIMTransportOption {
	return func(t *SWIMTransport) {
		t.streams = streams
	}
}

// SWIMPackets packet transports.
func SWIMPackets(packets ...net.PacketConn) SWIMTransportOption {
	return func(t *SWIMTransport) {
		t.packets = packets
	}
}

type dialer interface {
	DialContext(ctx context.Context, network, address string) (net.Conn, error)
}

// SWIMTransport is a Transport implementation that uses connectionless UDP for
// packet operations, and ad-hoc TCP connections for stream operations.
type SWIMTransport struct {
	dialer
	packetCh chan *memberlist.Packet
	streamCh chan net.Conn
	logger   *log.Logger
	wg       sync.WaitGroup
	streams  []net.Listener
	packets  []net.PacketConn
	shutdown int32
}

// NewSWIMTransport returns a net transport with the given configuration. On
// success all the network listeners will be created and listening.
func NewSWIMTransport(d dialer, options ...SWIMTransportOption) (*SWIMTransport, error) {
	// // If we reject the empty list outright we can assume that there's at
	// // least one listener of each type later during operation.

	// Build out the new transport.
	var ok bool
	t := SWIMTransport{
		dialer:   d,
		packetCh: make(chan *memberlist.Packet),
		streamCh: make(chan net.Conn),
		logger:   log.New(ioutil.Discard, "", 0),
	}

	for _, opt := range options {
		opt(&t)
	}

	if t.dialer == nil {
		return nil, fmt.Errorf("missing dialer")
	}
	if len(t.streams) == 0 {
		return nil, fmt.Errorf("At least one reliable transport (tcp, unix domain socket, net.Listener) required")
	}

	if len(t.packets) == 0 {
		return nil, fmt.Errorf("At least one unreliable packet transport (udp, net.PacketConn) required")
	}

	// Clean up listeners if there's an error.
	defer func() {
		if !ok {
			t.Shutdown()
		}
	}()

	// Fire them up now that we've been able to create them all.
	for idx := range t.streams {
		t.wg.Add(1)
		go t.tcpListen(t.streams[idx])
	}

	for idx := range t.streams {
		t.wg.Add(1)
		go t.udpListen(t.packets[idx])
	}

	ok = true
	return &t, nil
}

// GetAutoBindPort returns the bind port that was automatically given by the
// kernel, if a bind port of 0 was given.
func (t *SWIMTransport) GetAutoBindPort() int {
	// We made sure there's at least one TCP listener, and that one's
	// port was applied to all the others for the dynamic bind case.
	return t.packets[0].LocalAddr().(*net.UDPAddr).Port
}

// FinalAdvertiseAddr see memberlist.Transport.
func (t *SWIMTransport) FinalAdvertiseAddr(ip string, port int) (_ net.IP, _ int, err error) {
	var advertiseAddr net.IP
	var advertisePort int
	log.Println("FinalizeAdvertiseAddr", ip, port)

	if ip != "" {
		// If they've supplied an address, use that.
		advertiseAddr = net.ParseIP(ip)
		if advertiseAddr == nil {
			return nil, 0, fmt.Errorf("Failed to parse advertise address %q", ip)
		}

		// Ensure IPv4 conversion if necessary.
		if ip4 := advertiseAddr.To4(); ip4 != nil {
			advertiseAddr = ip4
		}
		advertisePort = port
	} else {
		switch s := t.streams[0].Addr().(type) {
		case *net.TCPAddr:
			if s.IP.IsUnspecified() {
				ip, err = sockaddr.GetPrivateIP()
				if err != nil {
					return nil, 0, fmt.Errorf("Failed to get interface addresses: %v", err)
				}
				if ip == "" {
					return nil, 0, fmt.Errorf("No private IP address found, and explicit IP not provided")
				}

				advertiseAddr = net.ParseIP(ip)
				if advertiseAddr == nil {
					return nil, 0, fmt.Errorf("Failed to parse advertise address: %q", ip)
				}
			} else {
				advertiseAddr = s.IP
			}
		default:
			return nil, 0, errors.Errorf("unknown network type: %T unable to determine IP/Port", s)
		}

		// Use the port we are bound to.
		advertisePort = t.GetAutoBindPort()
	}

	return advertiseAddr, advertisePort, nil
}

// WriteTo see Transport.
func (t *SWIMTransport) WriteTo(b []byte, addr string) (time.Time, error) {
	udpAddr, err := net.ResolveUDPAddr("udp", addr)
	if err != nil {
		return time.Time{}, err
	}

	// We made sure there's at least one UDP listener, so just use the
	// packet sending interface on the first one. Take the time after the
	// write call comes back, which will underestimate the time a little,
	// but help account for any delays before the write occurs.
	_, err = t.packets[0].WriteTo(b, udpAddr)
	return time.Now(), err
}

// PacketCh see memberlist.Transport.
func (t *SWIMTransport) PacketCh() <-chan *memberlist.Packet {
	return t.packetCh
}

// DialTimeout see memberlist.Transport.
func (t *SWIMTransport) DialTimeout(addr string, timeout time.Duration) (net.Conn, error) {
	ctx, done := context.WithTimeout(context.Background(), timeout)
	defer done()

	return t.dialer.DialContext(ctx, "tcp", addr)
}

// StreamCh see memberlist.Transport.
func (t *SWIMTransport) StreamCh() <-chan net.Conn {
	return t.streamCh
}

// Shutdown see memberlist.Transport.
func (t *SWIMTransport) Shutdown() error {
	// This will avoid log spam about errors when we shut down.
	atomic.StoreInt32(&t.shutdown, 1)

	// Rip through all the connections and shut them down.
	for _, conn := range t.streams {
		conn.Close()
	}

	for _, conn := range t.packets {
		conn.Close()
	}

	// Block until all the listener threads have died.
	t.wg.Wait()

	return nil
}

// tcpListen is a long running goroutine that accepts incoming TCP connections
// and hands them off to the stream channel.
func (t *SWIMTransport) tcpListen(l net.Listener) {
	defer t.wg.Done()
	limit := rate.NewLimiter(rate.Every(time.Second), 5)

	for {
		conn, err := l.Accept()
		if err != nil {
			if s := atomic.LoadInt32(&t.shutdown); s == 1 {
				break
			}

			if cause := limit.Wait(context.Background()); cause != nil {
				t.logger.Printf("[WARN] memberlist: accept rate failure %v", err)
			}

			t.logger.Printf("[ERR] memberlist: unable to accept connection: %v", err)
			continue
		}

		// No error, reset loop delay
		// loopDelay = 0

		t.streamCh <- conn
	}
}

// udpListen is a long running goroutine that accepts incoming UDP packets and
// hands them off to the packet channel.
func (t *SWIMTransport) udpListen(udpLn net.PacketConn) {
	defer t.wg.Done()
	for {
		// Do a blocking read into a fresh buffer. Grab a time stamp as
		// close as possible to the I/O.
		buf := make([]byte, udpPacketBufSize)
		n, addr, err := udpLn.ReadFrom(buf)
		ts := time.Now()
		if err != nil {
			if s := atomic.LoadInt32(&t.shutdown); s == 1 {
				break
			}

			t.logger.Printf("[ERR] memberlist: Error reading UDP packet: %v", err)
			continue
		}

		// Check the length - it needs to have at least one byte to be a
		// proper message.
		if n < 1 {
			t.logger.Printf("[ERR] memberlist: UDP packet too short (%d bytes) %s",
				len(buf), addr.String())
			continue
		}

		t.packetCh <- &memberlist.Packet{
			Buf:       buf[:n],
			From:      addr,
			Timestamp: ts,
		}
	}
}

// // setUDPRecvBuf is used to resize the UDP receive window. The function
// // attempts to set the read buffer to `udpRecvBuf` but backs off until
// // the read buffer can be set.
// func setUDPRecvBuf(c *net.UDPConn) error {
// 	size := udpRecvBufSize
// 	var err error
// 	for size > 0 {
// 		if err = c.SetReadBuffer(size); err == nil {
// 			return nil
// 		}
// 		size = size / 2
// 	}
// 	return err
// }
