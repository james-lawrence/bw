package dialers

import "google.golang.org/grpc"

// NewDefaults create a defaults dialer.
func NewDefaults(options ...grpc.DialOption) Defaults {
	return Defaulted(DefaultDialerOptions(options...))
}

type Defaulted []grpc.DialOption

func (t Defaulted) Defaults(combined ...grpc.DialOption) Defaulted {
	return append(t, combined...)
}
