package dialers

import (
	"context"

	"google.golang.org/grpc"
)

// NewDirect dials the provided address every time.
func NewDirect(address string, defaults ...grpc.DialOption) Direct {
	return Direct{
		address:  address,
		defaults: defaults,
	}
}

// Direct ...
type Direct struct {
	address  string
	defaults []grpc.DialOption
}

// Dial given the options
func (t Direct) Dial(options ...grpc.DialOption) (c *grpc.ClientConn, err error) {
	return t.DialContext(context.Background(), options...)
}

// DialContext given the context and options
func (t Direct) DialContext(ctx context.Context, options ...grpc.DialOption) (c *grpc.ClientConn, err error) {
	return grpc.DialContext(ctx, t.address, t.Defaults(options...)...)
}

// Defaults return the defaults for this dialer.
func (t Direct) Defaults(options ...grpc.DialOption) Defaulted {
	return append(t.defaults, options...)
}
