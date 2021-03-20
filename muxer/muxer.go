package muxer

import (
	"context"
	"crypto/md5"
	"io"
	"log"
	"net"
	sync "sync"
	"time"

	"github.com/hashicorp/yamux"
	"github.com/james-lawrence/bw/internal/x/errorsx"
	"github.com/pkg/errors"
	"google.golang.org/protobuf/proto"
)

type option func(*M) error

func New(options ...option) *M {
	return &M{
		m:         &sync.RWMutex{},
		protocols: make(map[Protocol]listener, 10),
	}
}

type M struct {
	m         *sync.RWMutex
	protocols map[Protocol]listener
}

func (t *M) Bind(protocol string, addr net.Addr) (net.Listener, error) {
	t.m.Lock()
	defer t.m.Unlock()

	digested := md5.Sum([]byte(protocol))
	if l, ok := t.protocols[digested]; ok {
		return l, errors.Errorf("protocol already registered: %s", protocol)
	}

	l := newListener(t, addr, digested)
	t.protocols[digested] = l

	return l, nil
}

func (t *M) release(p Protocol) {
	t.m.Lock()
	defer t.m.Unlock()

	delete(t.protocols, p)
}

func Listen(ctx context.Context, m *M, l net.Listener) error {
	for {
		var (
			err     error
			conn    net.Conn
			session *yamux.Session
			stream  net.Conn
			req     Protocol
		)

		if conn, err = l.Accept(); err != nil {
			return err
		}

		if session, err = yamux.Server(conn, nil); err != nil {
			return err
		}

		if stream, err = session.Accept(); err != nil {
			return err
		}

		if req, err = handshakeInbound(m, stream); err != nil {
			log.Println("inbound handshake failure", err)
			continue
		}

		m.m.RLock()
		protocol := m.protocols[req]
		m.m.RUnlock()

		select {
		case protocol.inbound <- stream:
		case <-ctx.Done():
			return ctx.Err()
		case <-time.After(time.Second):
			stream.Close()
			log.Println("accept failed - protocol took too long accept connection")
			continue
		}
	}
}

func handshakeOutbound(protocol []byte, conn net.Conn) (err error) {
	var (
		inbound [22]byte // 4 (version) + 4 (error) + protocol (36)
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
