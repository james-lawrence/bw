package cluster

import (
	"fmt"
	"log"
	"net"

	"bitbucket.org/jatone/bearded-wookie/clustering"
	"bitbucket.org/jatone/bearded-wookie/clustering/raftutil"
	"bitbucket.org/jatone/bearded-wookie/deployment/agent"
	"bitbucket.org/jatone/bearded-wookie/x/debugx"
	"github.com/golang/protobuf/proto"
	"github.com/hashicorp/memberlist"
	"github.com/pkg/errors"
)

// New ...
func New(l Local, c clustering.Cluster, s []byte) Cluster {
	return Cluster{
		local:        l,
		SharedSecret: s,
		Cluster:      c,
	}
}

// Cluster - represents a cluster.
type Cluster struct {
	clustering.Cluster
	SharedSecret []byte
	local        Local
}

// Local ...
func (t Cluster) Local() agent.Peer {
	return t.local.Peer
}

// Peers ...
func (t Cluster) Peers() []agent.Peer {
	return NodesToPeers(t.Cluster.Members()...)
}

// Details details about the cluster.
func (t Cluster) Details() agent.Details {
	quorum := NodesToPeers(raftutil.Quorum(t.Cluster)...)

	return agent.Details{
		Secret: t.SharedSecret,
		Quorum: PeersToPointers(quorum...),
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

// RPCAddress for peer.
func RPCAddress(p agent.Peer) string {
	return net.JoinHostPort(p.Ip, fmt.Sprint(p.RPCPort))
}

// NodeRPCAddress returns the node's rpc address.
// if an error occurs it returns a blank string.
func NodeRPCAddress(n *memberlist.Node) string {
	var (
		err error
		p   agent.Peer
	)

	if p, err = NodeToPeer(n); err != nil {
		debugx.Println("failed to convert node to peer", err)
		return ""
	}

	return RPCAddress(p)
}
