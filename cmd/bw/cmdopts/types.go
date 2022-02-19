package cmdopts

import (
	"net"
	"reflect"
	"strings"

	"github.com/alecthomas/kong"
	"github.com/pkg/errors"
)

// ParseIP addresses
func ParseIP(ctx *kong.DecodeContext, target reflect.Value) (err error) {
	target.Set(reflect.ValueOf(net.ParseIP(ctx.Scan.Pop().String())))
	return nil
}

func ParseTCPAddr(ctx *kong.DecodeContext, target reflect.Value) (err error) {
	var (
		saddr = ctx.Scan.Pop().String()
	)

	if ctx.Scan.Len() == 0 {
		return nil
	}

	var (
		addr *net.TCPAddr
	)

	if addr, err = net.ResolveTCPAddr("tcp", saddr); err != nil {
		return errors.Wrapf(err, "unable to resolve tcp address %s", saddr)
	}

	target.Set(reflect.ValueOf(addr))

	return nil
}

func ParseTCPAddrArray(ctx *kong.DecodeContext, target reflect.Value) (err error) {
	var (
		results []*net.TCPAddr
	)

	if ctx.Scan.Len() == 0 {
		return nil
	}

	for _, saddr := range strings.Split(ctx.Scan.Pop().String(), "\n") {
		var (
			addr *net.TCPAddr
		)

		if addr, err = net.ResolveTCPAddr("tcp", saddr); err != nil {
			return errors.Wrapf(err, "unable to resolve tcp address %s", saddr)
		}

		results = append(results, addr)
	}

	target.Set(reflect.ValueOf(results))
	return nil
}
