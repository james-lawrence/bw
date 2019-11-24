package discovery

import (
	"context"

	"google.golang.org/grpc"
)

// New service.
func New(c cluster) Discovery {
	return Discovery{
		c: c,
	}
}

// Discovery provides information requestors
type Discovery struct {
	c cluster
}

// Bind the service to the given grpc server.
func (t Discovery) Bind(s *grpc.Server) {
	RegisterDiscoveryServer(s, t)
}

// Quorum returns information about the quorum nodes.
func (t Discovery) Quorum(ctx context.Context, req *QuorumRequest) (resp *QuorumResponse, err error) {
	resp = &QuorumResponse{}

	for _, p := range t.c.Quorum() {
		n := peerToNode(p)
		resp.Nodes = append(resp.Nodes, &n)
	}

	return resp, err
}
