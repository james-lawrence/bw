package dialers

import (
	"context"

	"google.golang.org/grpc"
)

// NewQuorum dialer.
func NewQuorum(c rendezvous, defaults ...grpc.DialOption) Quorum {
	return Quorum{defaults: NewDefaults(defaults...).Defaults(grpc.WithResolvers(NewRingResolver(c)))}
}

// Quorum dialers members of Quorum.
type Quorum struct {
	defaults []grpc.DialOption
}

// DialContext with the given options
func (t Quorum) DialContext(ctx context.Context, options ...grpc.DialOption) (c *grpc.ClientConn, err error) {
	return grpc.DialContext(ctx, quorumScheme+"://", t.Defaults()...)
}

// Defaults return the defaults for this dialer.
func (t Quorum) Defaults(options ...grpc.DialOption) Defaulted {
	return NewDefaults(t.defaults...).Defaults(options...)
}
