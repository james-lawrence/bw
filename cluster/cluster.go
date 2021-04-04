package cluster

import (
	"github.com/hashicorp/memberlist"
	"github.com/james-lawrence/bw/agent"
)

type rendezvous interface {
	Get([]byte) *memberlist.Node
	GetN(n int, key []byte) []*memberlist.Node
}

type cluster interface {
	Members() []*memberlist.Node
	Get([]byte) *memberlist.Node
	GetN(n int, key []byte) []*memberlist.Node
	LocalNode() *memberlist.Node
}

// New ...
func New(l *Local, c cluster) Cluster {
	return Cluster{
		local:   l,
		cluster: c,
	}
}

// Cluster - represents a cluster.
type Cluster struct {
	cluster
	local *Local
}

// Local ...
func (t Cluster) Local() *agent.Peer {
	return t.local.Peer
}

// Peers ...
func (t Cluster) Peers() []*agent.Peer {
	return agent.NodesToPeers(t.cluster.Members()...)
}

// Quorum ...
func (t Cluster) Quorum() []*agent.Peer {
	return agent.QuorumPeers(t)
}

// Connect connection information for the cluster.
func (t Cluster) Connect() agent.ConnectResponse {
	return agent.ConnectResponse{
		Quorum: t.Quorum(),
	}
}
