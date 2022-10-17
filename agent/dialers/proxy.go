package dialers

import (
	"context"
	"log"

	"github.com/james-lawrence/bw/agent"
	"github.com/pkg/errors"
	"google.golang.org/grpc"
)

// NewProxy dialer allows for dialer quorum by using a proxy agent.
func NewProxy(d DefaultsDialer) Proxy {
	return Proxy{d: d}
}

// Proxy dialers members of quorum.
type Proxy struct {
	d DefaultsDialer
}

// DialContext with the given options
func (t Proxy) DialContext(ctx context.Context, options ...grpc.DialOption) (conn *grpc.ClientConn, err error) {
	var (
		proxy *grpc.ClientConn
		cinfo *agent.ConnectResponse
	)

	if proxy, err = t.d.DialContext(ctx, options...); err != nil {
		return nil, err
	}
	defer proxy.Close()

	if cinfo, err = agent.NewConn(proxy).Connect(ctx); err != nil {
		return conn, err
	}

	for _, q := range agent.Shuffle(cinfo.Quorum) {
		addr := agent.RPCAddress(q)
		if conn, err = grpc.DialContext(ctx, addr, t.d.Defaults(options...)...); err != nil {
			log.Println("failed to dial", addr, err)
			continue
		}

		return conn, nil
	}

	return nil, errors.WithMessage(err, "failed to connect to a member of the quorum")
}
