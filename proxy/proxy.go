package proxy

import (
	"context"
	"encoding/binary"
	"errors"
	"io"
	"log"
	"net"
)

// Proxy copies data between client and dst until either side finishes, then closes both.
func Proxy(ctx context.Context, client, dst net.Conn, buf io.Reader) error {
	var (
		errors = make(chan error, 2)
	)

	log.Println("proxying connection")
	ctx, done := context.WithCancel(ctx)
	defer done()
	defer client.Close()
	defer dst.Close()

	go func() {
		errors <- proxyConn(ctx, done, client, dst, buf)
	}()
	go func() {
		errors <- proxyConn(ctx, done, dst, client, nil)
	}()

	// the first side to finish (or the context) ends the session, the deferred closes
	// unblock the other copy.
	select {
	case err := <-errors:
		return err
	case <-ctx.Done():
		return ctx.Err()
	}
}

func proxyConn(ctx context.Context, done context.CancelFunc, from, to net.Conn, buf io.Reader) (err error) {
	defer done()

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

// MaxWireformat is the largest message WireformatDecode accepts. the length prefix is read
// before the peer is authenticated, so it must not be trusted to size an allocation.
const MaxWireformat = 1 << 20

// ErrWireformatTooLarge is returned when the length prefix exceeds MaxWireformat.
var ErrWireformatTooLarge = errors.New("proxy: message exceeds the maximum size")

func WireformatDecode(src io.Reader) (buf []byte, err error) {
	buf = make([]byte, 8)

	if _, err = io.ReadFull(src, buf); err != nil {
		return buf, err
	}

	length := binary.LittleEndian.Uint64(buf)
	if length > MaxWireformat {
		return nil, ErrWireformatTooLarge
	}

	buf = make([]byte, length)

	if _, err = io.ReadFull(src, buf); err != nil {
		return buf, err
	}

	return buf, err
}
