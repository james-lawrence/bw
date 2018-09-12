package cluster

import (
	"time"

	"github.com/hashicorp/memberlist"
	"github.com/hashicorp/raft"
	"github.com/james-lawrence/bw/agent"
	"github.com/james-lawrence/bw/clustering/raftutil"
)

type cluster interface {
	Leave(time.Duration) error
	Join(...string) (int, error)
	Members() []*memberlist.Node
	Get([]byte) *memberlist.Node
	GetN(n int, key []byte) []*memberlist.Node
	LocalNode() *memberlist.Node
	Config() *memberlist.Config
}

// New ...
func New(l Local, c cluster) Cluster {
	return Cluster{
		local:   l,
		cluster: c,
	}
}

// Cluster - represents a cluster.
type Cluster struct {
	cluster
	local Local
}

// Leave ...
func (t Cluster) Leave() error {
	return t.cluster.Leave(5 * time.Second)
}

// Join ...
func (t Cluster) Join(peers ...string) (int, error) {
	return t.cluster.Join(peers...)
}

// Local ...
func (t Cluster) Local() agent.Peer {
	return t.local.Peer
}

// Peers ...
func (t Cluster) Peers() []agent.Peer {
	return agent.NodesToPeers(t.cluster.Members()...)
}

// Quorum ...
func (t Cluster) Quorum() []agent.Peer {
	return agent.NodesToPeers(raftutil.Quorum(3, t.cluster)...)
}

// Connect connection information for the cluster.
func (t Cluster) Connect() agent.ConnectResponse {
	var (
		secret []byte
	)

	if c := t.cluster.Config(); c != nil {
		secret = c.Keyring.GetPrimaryKey()
	}

	return agent.ConnectResponse{
		Secret: secret,
		Quorum: agent.PeersToPtr(t.Quorum()...),
	}
}

// NewRaftAddressProvider converts memberlist.Node into a raft.Server
func NewRaftAddressProvider(c cluster) RaftAddressProvider {
	return RaftAddressProvider{
		cluster: c,
	}
}

// RaftAddressProvider ...
type RaftAddressProvider struct {
	cluster
}

// RaftAddr ...
func (t RaftAddressProvider) RaftAddr(n *memberlist.Node) (_zero raft.Server, err error) {
	var (
		p agent.Peer
	)

	if p, err = agent.NodeToPeer(n); err != nil {
		return _zero, err
	}

	return raft.Server{
		ID:       raft.ServerID(n.Name),
		Address:  raft.ServerAddress(agent.RaftAddress(p)),
		Suffrage: raft.Voter,
	}, err
}
