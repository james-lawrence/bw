package dialers

import (
	"net"
	"time"

	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/keepalive"
)

func clonedefault(options ...grpc.DialOption) Defaulted {
	var copied = make([]grpc.DialOption, len(options))
	copy(copied, options)
	return copied
}

// NewDefaults create a defaults dialer.
func NewDefaults(options ...grpc.DialOption) Defaults {
	return Defaulted(options)
}

type Defaulted []grpc.DialOption

func (t Defaulted) Defaults(combined ...grpc.DialOption) Defaulted {
	return append(clonedefault(t...), combined...)
}

func EndpointDialer(address string, d dialer, options ...grpc.DialOption) (_d Defaults, err error) {
	var (
		addr *net.TCPAddr
	)

	if addr, err = net.ResolveTCPAddr("tcp", address); err != nil {
		return _d, err
	}

	return NewDefaults(
		WithMuxer(
			d,
			addr,
		),
		// insecure credentials is safe here because WithMuxer uses tls.
		grpc.WithTransportCredentials(insecure.NewCredentials()),
		grpc.WithKeepaliveParams(keepalive.ClientParameters{
			Time: 15 * time.Second,
		}),
	).Defaults(
		options...,
	), nil
}
