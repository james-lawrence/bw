package agent

import (
	"fmt"
	"net"

	"github.com/james-lawrence/bw/x/systemx"
)

// RPCAddress for peer.
func RPCAddress(p Peer) string {
	return net.JoinHostPort(p.Ip, fmt.Sprint(p.RPCPort))
}

// SWIMAddress for peer.
func SWIMAddress(p Peer) string {
	return net.JoinHostPort(p.Ip, fmt.Sprint(p.SWIMPort))
}

// RaftAddress for peer.
func RaftAddress(p Peer) string {
	return net.JoinHostPort(p.Ip, fmt.Sprint(p.RaftPort))
}

// LocalPeer build local peer.
func LocalPeer(id string) Peer {
	return Peer{
		Name:     id,
		Ip:       systemx.HostIP(systemx.HostnameOrLocalhost()).String(),
		RPCPort:  2000,
		SWIMPort: 2001,
		RaftPort: 2002,
		Status:   Peer_Ready,
	}
}

// PeersToPtr util function to convert between pointers and values.
func PeersToPtr(peers ...Peer) []*Peer {
	r := make([]*Peer, 0, len(peers))

	for _, p := range peers {
		tmp := p
		r = append(r, &tmp)
	}

	return r
}

// PtrToPeers util function to convert between pointers and values.
func PtrToPeers(peers ...*Peer) []Peer {
	r := make([]Peer, 0, len(peers))

	for _, p := range peers {
		r = append(r, *p)
	}

	return r
}
