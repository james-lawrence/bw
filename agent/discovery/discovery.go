// Package discovery is used to provide information
// about the system to anyone.
package discovery

import (
	"context"

	"github.com/hashicorp/memberlist"
	"github.com/pkg/errors"
	"google.golang.org/grpc"

	"github.com/james-lawrence/bw/agent"
	"github.com/james-lawrence/bw/agent/dialers"
	"github.com/james-lawrence/bw/internal/systemx"
)

// cluster interface for the package.
type cluster interface {
	Quorum() []*agent.Peer
	Peers() []*agent.Peer
}

func peerToNode(p *agent.Peer) Node {
	return Node{
		Ip:      p.Ip,
		Name:    p.Name,
		P2PPort: p.P2PPort,
	}
}

// nodeToPeer ...
func nodeToPeer(n *Node) *agent.Peer {
	return &agent.Peer{
		Ip:      n.Ip,
		Name:    n.Name,
		P2PPort: n.P2PPort,
	}
}

func nodesToMembers(ns ...*Node) (r []*memberlist.Node) {
	for _, n := range ns {
		r = append(r, agent.PeerToNode(nodeToPeer(n)))
	}
	return r
}

func defaultAgentsRequest() *AgentsRequest {
	return &AgentsRequest{Maximum: 100}
}

// Snapshot ...
func Snapshot(address string, options ...grpc.DialOption) (nodes []*memberlist.Node, err error) {
	var (
		cc *grpc.ClientConn
		s  Discovery_AgentsClient
	)

	if cc, err = dialers.NewDirect(address).Dial(options...); err != nil {
		return nodes, err
	}
	defer cc.Close()

	if s, err = NewDiscoveryClient(cc).Agents(context.Background(), defaultAgentsRequest()); err != nil {
		return nodes, err
	}

	for batch, err := s.Recv(); err == nil; batch, err = s.Recv() {
		nodes = append(nodes, nodesToMembers(batch.Nodes...)...)
	}

	return nodes, err
}

// CheckCredentials against discovery
func CheckCredentials(address string, path string, d dialers.Defaults) (err error) {
	var (
		cc *grpc.ClientConn
	)

	if !systemx.FileExists(path) {
		return nil
	}

	fingerprint := systemx.FileMD5(path)
	if fingerprint == "" {
		return errors.New("failed to generate fingerprint")
	}

	if cc, err = dialers.NewDirect(agent.URIAgent(address)).Dial(d.Defaults()...); err != nil {
		return err
	}
	defer cc.Close()

	_, err = NewAuthorityClient(cc).Check(context.Background(), &CheckRequest{Fingerprint: fingerprint})
	return err
}
