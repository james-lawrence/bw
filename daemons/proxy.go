package daemons

import (
	"context"
	"log"
	"net"

	"github.com/james-lawrence/bw/agent/discovery"
)

type proxydialer interface {
	DialContext(ctx context.Context, network string, address string) (net.Conn, error)
}

func Proxy(dctx Context, d proxydialer) (_ Context, err error) {
	var (
		l net.Listener
	)

	if l, err = dctx.Muxer.Bind("bw.proxy", dctx.Listener.Addr()); err != nil {
		return dctx, err
	}

	go func() {
		if err = discovery.Proxy(l, d); err != nil {
			log.Println("proxy failed", err)
		}
	}()

	return dctx, nil
}
