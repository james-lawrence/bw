package daemons

import (
	"context"
	"log"
	"net"

	"github.com/james-lawrence/bw"
	"github.com/james-lawrence/bw/agent/discovery"
	"github.com/james-lawrence/bw/notary"
	"github.com/pkg/errors"
)

type proxydialer interface {
	DialContext(ctx context.Context, network string, address string) (net.Conn, error)
}

func Proxy(dctx Context, d proxydialer) (_ Context, err error) {
	var (
		l net.Listener
	)

	if l, err = dctx.Muxer.Bind(bw.ProtocolProxy, dctx.Listener.Addr()); err != nil {
		return dctx, err
	}

	go func() {
		if err = discovery.Proxy(l, d, notary.NewAuthChecker(dctx.NotaryStorage, func(perm *notary.Permission) (err error) {
			if perm.Deploy {
				return nil
			}

			return errors.New("unauthorized")
		})); err != nil {
			log.Println("proxy failed", err)
		}
	}()

	return dctx, nil
}
