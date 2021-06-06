package proxy

import (
	"context"
	"encoding/binary"
	"io"
	"log"
	"net"

	"github.com/james-lawrence/bw/internal/x/errorsx"
)

type dialer interface {
	DialContext(ctx context.Context, network string, address string) (net.Conn, error)
}

func Proxy(ctx context.Context, c1, c2 net.Conn, buf io.Reader) error {
	var (
		errors chan error
	)

	ctx, done := context.WithCancel(ctx)
	go func() {
		select {
		case errors <- proxyConn(ctx, done, c1, c2, buf):
		case <-ctx.Done():
			errors <- ctx.Err()
		}
	}()
	go func() {
		select {
		case errors <- proxyConn(ctx, done, c2, c1, nil):
		case <-ctx.Done():
			errors <- ctx.Err()
		}
	}()

	return errorsx.Compact(<-errors, <-errors)
}

func proxyConn(ctx context.Context, done context.CancelFunc, from, to net.Conn, buf io.Reader) (err error) {
	defer done()
	defer log.Println("proxy conn completed")

	if buf != nil {
		if _, err = io.Copy(to, buf); err != nil {
			return err
		}
	}

	if _, err = io.Copy(to, from); err != nil {
		return err
	}

	return nil
}

func WireformatEncode(encoded []byte) []byte {
	var (
		buf = make([]byte, 8)
	)

	binary.LittleEndian.PutUint64(buf, uint64(len(encoded)))

	return append(buf, encoded...)
}

func WireformatDecode(src io.Reader) (buf []byte, err error) {
	buf = make([]byte, 8)

	if _, err = io.ReadFull(src, buf); err != nil {
		return buf, err
	}

	length := binary.LittleEndian.Uint64(buf)
	buf = make([]byte, length)

	if _, err = io.ReadFull(src, buf); err != nil {
		return buf, err
	}

	return buf, err
}
