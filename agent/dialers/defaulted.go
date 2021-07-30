package dialers

import (
	"net"

	"google.golang.org/grpc"
)

// NewDefaults create a defaults dialer.
func NewDefaults(options ...grpc.DialOption) Defaults {
	return Defaulted(DefaultDialerOptions(options...))
}

type Defaulted []grpc.DialOption

func (t Defaulted) Defaults(combined ...grpc.DialOption) Defaulted {
	return append(t, combined...)
}

func DefaultDialer(address string, d dialer, options ...grpc.DialOption) (_d Defaults, err error) {
	var (
		addr *net.TCPAddr
	)

	if addr, err = net.ResolveTCPAddr("tcp", address); err != nil {
		return _d, err
	}

	return NewDefaults(options...).Defaults(
		WithMuxer(
			d,
			addr,
		),
		grpc.WithInsecure(),
	), nil
}
