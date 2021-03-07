package dialers

import "google.golang.org/grpc"

// NewDefaults create a defaults dialer.
func NewDefaults(options ...grpc.DialOption) Defaults {
	return defaulted(DefaultDialerOptions(options...))
}

type defaulted []grpc.DialOption

func (t defaulted) Defaults(combined ...grpc.DialOption) []grpc.DialOption {
	return append(t, combined...)
}
