package cluster

import (
	"log"

	"github.com/james-lawrence/bw/agent"
	"github.com/james-lawrence/bw/clustering/raftutil"
	"github.com/golang/protobuf/proto"
	"github.com/hashicorp/memberlist"
	"github.com/pkg/errors"
)

type cluster interface {
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

// Local ...
func (t Cluster) Local() agent.Peer {
	return t.local.Peer
}

// Peers ...
func (t Cluster) Peers() []agent.Peer {
	return NodesToPeers(t.cluster.Members()...)
}

// Quorum ...
func (t Cluster) Quorum() []agent.Peer {
	return NodesToPeers(raftutil.Quorum(t.cluster)...)
}

// Connect connection information for the cluster.
func (t Cluster) Connect() agent.ConnectInfo {
	var (
		secret []byte
	)

	if c := t.cluster.Config(); c != nil {
		secret = c.SecretKey
	}

	return agent.ConnectInfo{
		Secret: secret,
		Quorum: agent.PeersToPtr(t.Quorum()...),
	}
}

// NodesToPeers ...
func NodesToPeers(nodes ...*memberlist.Node) []agent.Peer {
	peers := make([]agent.Peer, 0, len(nodes))
	for _, n := range nodes {
		var (
			err  error
			peer agent.Peer
		)

		if peer, err = NodeToPeer(n); err != nil {
			log.Println("skipping node", n.Name, "invalid metadata", err)
		}

		peers = append(peers, peer)
	}

	return peers
}

// NodeToPeer converts a node to a peer
func NodeToPeer(n *memberlist.Node) (_zerop agent.Peer, err error) {
	var (
		m Metadata
	)

	if err = proto.Unmarshal(n.Meta, &m); err != nil {
		return _zerop, errors.WithStack(err)
	}

	return agent.Peer{
		Status:   agent.Peer_Unknown,
		Name:     n.Name,
		Ip:       n.Addr.String(),
		RPCPort:  m.RPCPort,
		SWIMPort: m.SWIMPort,
		RaftPort: m.RaftPort,
	}, nil
}
