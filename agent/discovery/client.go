package discovery

import (
	"context"

	"github.com/james-lawrence/bw/agent"
	"google.golang.org/grpc"
)

// NewQuorumDialer use the address to build a quorum dialer
func NewQuorumDialer(address string) (QuorumDialer, error) {
	conn, err := grpc.Dial(address)
	return QuorumDialer{
		conn:    conn,
		address: agent.RPCAddress,
	}, err
}

// QuorumDialer ...
type QuorumDialer struct {
	conn    *grpc.ClientConn
	address func(agent.Peer) string
}

// Dial given the options
func (t QuorumDialer) Dial(options ...grpc.DialOption) (c *grpc.ClientConn, err error) {
	var (
		resp *QuorumResponse
	)

	if resp, err = NewDiscoveryClient(t.conn).Quorum(context.Background(), &QuorumRequest{}); err != nil {
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
