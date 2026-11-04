package agent

import (
	"math/rand"

	"github.com/hashicorp/memberlist"
	"github.com/james-lawrence/bw/clustering/rendezvous"
)

type rendez interface {
	GetN(int, []byte) []*memberlist.Node
}

// Random returns a single random peer from the set.
// if the set is empty then a zero value peer is returned.
func Random(peers ...*Peer) (p *Peer) {
	for _, p = range shuffleQuorum(peers) {
		return p
	}

	return p
}

// Shuffle the peers
func Shuffle(q []*Peer) []*Peer {
	return shuffleQuorum(q)
}

// QuorumPeers helper method.
func QuorumPeers(c rendez) []*Peer {
	return Shuffle(NodesToPeers(QuorumNodes(c)...))
}

// QuorumNodes return the quorum nodes.
func QuorumNodes(c rendez) []*memberlist.Node {
	return c.GetN(QuorumDefault, []byte(QuorumKey))
}

// LargeQuorum returns 2x the nodes required to achieve quorum.
func LargeQuorum(c rendez) []*memberlist.Node {
	return c.GetN(2*QuorumDefault, []byte(QuorumKey))
}

// DeploymentNode return the quorum nodes.
func DeploymentNode(c rendez, key []byte) []*memberlist.Node {
	return rendezvous.MaxN(1, key, QuorumNodes(c))
}

// RendezvousPeers based on the provided k generate the peers.
func RendezvousPeers(k []byte, c rendez) []*Peer {
	return Shuffle(NodesToPeers(c.GetN(2*QuorumDefault, k)...))
}

func shuffleQuorum(q []*Peer) []*Peer {
	rand.Shuffle(len(q), func(i int, j int) {
		q[i], q[j] = q[j], q[i]
	})
	return q
}
