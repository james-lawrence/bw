package discovery

import (
	"context"
	"sync/atomic"

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
	UnimplementedDiscoveryServer
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

// Agents returns nodes of the cluster.
func (t Discovery) Agents(req *AgentsRequest, dst Discovery_AgentsServer) (err error) {
	atomic.CompareAndSwapInt64(&req.Maximum, 0, 100)

	resp := &AgentsResponse{}

	for idx, p := range t.c.Peers() {
		n := peerToNode(p)
		resp.Nodes = append(resp.Nodes, &n)

		if int64(idx)%req.Maximum == 0 {
			if err = dst.Send(resp); err != nil {
				return err
			}

			resp = &AgentsResponse{}
		}
	}

	if len(resp.Nodes) > 0 {
		if err = dst.Send(resp); err != nil {
			return err
		}
	}

	return err
}
