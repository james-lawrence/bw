package cluster

import (
	"log"

	"bitbucket.org/jatone/bearded-wookie/agent"
	"bitbucket.org/jatone/bearded-wookie/clustering"
	"bitbucket.org/jatone/bearded-wookie/clustering/raftutil"
	"github.com/golang/protobuf/proto"
	"github.com/hashicorp/memberlist"
	"github.com/pkg/errors"
)

// New ...
func New(l Local, c clustering.Cluster) Cluster {
	return Cluster{
		local:   l,
		Cluster: c,
	}
}

// Cluster - represents a cluster.
type Cluster struct {
	clustering.Cluster
	local Local
}

// Local ...
func (t Cluster) Local() agent.Peer {
	return t.local.Peer
}

// Peers ...
func (t Cluster) Peers() []agent.Peer {
	return NodesToPeers(t.Cluster.Members()...)
}

// Quorum ...
func (t Cluster) Quorum() []agent.Peer {
	return NodesToPeers(raftutil.Quorum(t.Cluster)...)
}

// Connect connection information for the cluster.
func (t Cluster) Connect() agent.ConnectInfo {
	var (
		secret []byte
	)

	if c := t.Cluster.Config(); c != nil {
		secret = c.SecretKey
	}

	return agent.ConnectInfo{
		Secret: secret,
		Quorum: PeersToPointers(t.Quorum()...),
	}
}

// PeersToPointers converts a set of peers to a slice of pointers.
func PeersToPointers(peers ...agent.Peer) []*agent.Peer {
	pts := make([]*agent.Peer, 0, len(peers))
	for _, n := range peers {
		tmp := n
		pts = append(pts, &tmp)
	}

	return pts
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
