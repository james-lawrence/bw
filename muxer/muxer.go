package muxer

import (
	"context"
	"crypto/md5"
	"crypto/tls"
	"encoding/hex"
	"io"
	"log"
	"net"
	"runtime"
	sync "sync"
	"time"

	"github.com/james-lawrence/bw/internal/x/errorsx"
	"github.com/pkg/errors"
	"google.golang.org/protobuf/proto"
)

type option func(*M) error

func New(options ...option) *M {
	return &M{
		m:             &sync.RWMutex{},
		protocols:     make(map[Protocol]*listener, 10),
		acceptTimeout: time.Second,
	}
}

type M struct {
	m             *sync.RWMutex
	protocols     map[Protocol]*listener
	defaulted     *listener
	acceptTimeout time.Duration
}

func (t *M) bind(protocol string, addr net.Addr) (net.Listener, error) {
	digested := md5.Sum([]byte(protocol))
	if l, ok := t.protocols[digested]; ok {
		return l, errors.Errorf("protocol already registered: %s", protocol)
	}

	l := newListener(t, addr, protocol, digested)
	t.protocols[digested] = l

	// log.Println("BOUND", protocol, "->", hex.EncodeToString(digested[:]))

	return l, nil
}

func (t *M) Bind(protocol string, addr net.Addr) (net.Listener, error) {
	t.m.Lock()
	defer t.m.Unlock()

	return t.bind(protocol, addr)
}

func (t *M) Default(protocol string, addr net.Addr) (net.Listener, error) {
	t.m.Lock()
	defer t.m.Unlock()
	t.defaulted = newListener(t, addr, protocol, md5.Sum([]byte(protocol)))

	return t.defaulted, nil
}

func (t *M) release(p Protocol) {
	t.m.Lock()
	defer t.m.Unlock()

	delete(t.protocols, p)
}

func Listen(ctx context.Context, m *M, l net.Listener) error {
	inbound := make(chan net.Conn, 200)
	for i := 0; i < runtime.NumCPU(); i++ {
		go accept1(ctx, m, inbound)
	}

	// log.Println("spawned", runtime.NumCPU(), "accepts")

	for {
		var (
			err  error
			conn net.Conn
		)

		if conn, err = l.Accept(); err != nil {
			log.Println("accept failed", err)
			return err
		}

		// log.Printf("accept: initiated %T %p backlog(%d) cap(%d)\n", conn, inbound, len(inbound), cap(inbound))
		select {
		case inbound <- conn:
		case <-ctx.Done():
			conn.Close()
			return ctx.Err()
		}
	}
}

func accept1(ctx context.Context, m *M, inbound chan net.Conn) (err error) {
	for {
		select {
		case conn := <-inbound:
			// log.Printf("accept: completed %T %p backlog(%d) cap(%d)\n", conn, inbound, len(inbound), cap(inbound))
			if err = accept(ctx, m, conn); err != nil {
				conn.Close()
				// log.Println("accept failed", err)
			}
		case <-ctx.Done():
			log.Println("accept failed", ctx.Err())
			return ctx.Err()
		}
	}
}

func accept(ctx context.Context, m *M, conn net.Conn) (err error) {
	var (
		req Protocol
	)

	cctx, done := context.WithTimeout(ctx, m.acceptTimeout)
	defer done()
	// log.Println("accept initiated")
	// defer log.Println("accept completed")

	if tlsconn, ok := conn.(*tls.Conn); ok {
		if err = tlsconn.Handshake(); err != nil {
			conn.Close()
			return errors.Wrap(err, "tls handshake failed")
		}

		if s := tlsconn.ConnectionState(); s.NegotiatedProtocol != "bw.mux" {
			if m.defaulted == nil {
				conn.Close()
				return errors.Wrap(err, "tls unknown protocol")
			}

			m.defaulted.inbound <- conn
			return nil
		}
	}

	if req, err = handshakeInbound(m, conn); err != nil {
		conn.Close()
		return errors.Wrap(err, "muxer.handshakeInbound failed")
	}

	m.m.RLock()
	protocol, ok := m.protocols[req]
	m.m.RUnlock()

	if !ok {
		conn.Close()
		return errors.Errorf("unknown protocol: %s", hex.EncodeToString(req[:]))
	}

	// log.Println("muxer.Accept", protocol.protocol, conn.RemoteAddr().String(), "->", conn.LocalAddr().String())
	select {
	case protocol.inbound <- conn:
		return nil
	case <-cctx.Done():
		conn.Close()
		return cctx.Err()
	}
}

func handshakeOutbound(protocol []byte, conn net.Conn) (err error) {
	var (
		inbound [22]byte // 4 (version) + 2 (error) + protocol (16)
		resp    Accepted
	)

	conn.SetWriteDeadline(time.Now().Add(time.Second))
	defer conn.SetWriteDeadline(time.Time{})

	if err = req(conn, protocol); err != nil {
		return errorsx.Compact(err, conn.Close())
	}

	if n, err := io.ReadFull(conn, inbound[:]); err != nil {
		return err
	} else if n != len(inbound) {
		return err
	}

	if err = proto.Unmarshal(inbound[:], &resp); err != nil {
		return err
	}

	switch resp.Code {
	case Accepted_None:
		return nil
	default:
		return errors.Errorf("bad handshake: %s", resp.Code.String())
	}
}

func handshakeInbound(m *M, conn net.Conn) (protocol Protocol, err error) {
	var (
		unknown Protocol
		req     Requested
		inbound [20]byte // 4 (version) + protocol (16)
	)

	conn.SetReadDeadline(time.Now().Add(time.Second))
	defer conn.SetReadDeadline(time.Time{}) // remove deadline

	if n, err := io.ReadFull(conn, inbound[:]); err != nil {
		return unknown, errorsx.Compact(err, reject(conn, unknown[:], Accepted_ClientError))
	} else if n != len(inbound) {
		return unknown, errorsx.Compact(err, reject(conn, unknown[:], Accepted_ClientError))
	}

	if err = proto.Unmarshal(inbound[:], &req); err != nil {
		return unknown, errorsx.Compact(err, reject(conn, unknown[:], Accepted_ClientError))
	}

	copy(protocol[:], req.Protocol)

	return protocol, ack(conn, req.Protocol, Accepted_None)
}

func req(conn net.Conn, protocol []byte) (err error) {
	var (
		encoded []byte
	)

	encoded, err = proto.Marshal(&Requested{
		Version:  1,
		Protocol: protocol,
	})

	if err != nil {
		return err
	}

	if _, err = conn.Write(encoded); err != nil {
		return err
	}

	return nil
}

func reject(conn net.Conn, protocol []byte, code AcceptedError) (err error) {
	defer conn.Close()
	return ack(conn, protocol, code)
}

func ack(conn net.Conn, protocol []byte, code AcceptedError) (err error) {
	var (
		encoded []byte
	)

	encoded, err = proto.Marshal(&Accepted{
		Version:  1,
		Protocol: protocol,
		Code:     code,
	})

	if err != nil {
		return err
	}

	if _, err = conn.Write(encoded); err != nil {
		return err
	}

	return nil
}
