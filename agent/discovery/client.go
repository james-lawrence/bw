package discovery

import (
	"context"

	"github.com/james-lawrence/bw/agent"
	"github.com/james-lawrence/bw/internal/x/grpcx"
	"google.golang.org/grpc"
)

// NewQuorumDialer use the address to build a quorum dialer
func NewQuorumDialer(address string) (QuorumDialer, error) {
	return QuorumDialer{
		addr:    address,
		address: agent.RPCAddress,
		cached:  grpcx.NewCachedClient(),
	}, nil
}

// QuorumDialer ...
type QuorumDialer struct {
	addr    string
	cached  *grpcx.CachedClient
	address func(agent.Peer) string
}

// Dial given the options
func (t QuorumDialer) Dial(options ...grpc.DialOption) (c *grpc.ClientConn, err error) {
	var (
		proxy *grpc.ClientConn
		resp  *QuorumResponse
	)

	if proxy, err = t.cached.Dial(t.addr, options...); err != nil {
		return c, err
	}

	if resp, err = NewDiscoveryClient(proxy).Quorum(context.Background(), &QuorumRequest{}); err != nil {
		return c, err
	}

	for _, n := range resp.Nodes {
		address := t.address(nodeToPeer(*n))
		if c, err = grpc.Dial(address, options...); err == nil {
			return c, err
		}
	}

	return c, err
}
