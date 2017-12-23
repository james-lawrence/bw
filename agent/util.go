package agent

import (
	"fmt"
	"net"
	"strconv"

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

// StaticPeeringStrategy ...
func StaticPeeringStrategy(peers ...Peer) []string {
	results := make([]string, 0, len(peers))
	for _, p := range peers {
		results = append(results, SWIMAddress(p))
	}

	return results
}

// PeerOption ...
type PeerOption func(*Peer)

// PeerOptionIP set IP for peer.
func PeerOptionIP(ip net.IP) PeerOption {
	return func(p *Peer) {
		p.Ip = ip.String()
	}
}

// PeerOptionRPCPort ...
func PeerOptionRPCPort(port uint32) PeerOption {
	return func(p *Peer) {
		p.RPCPort = port
	}
}

// PeerOptionSWIMPort ...
func PeerOptionSWIMPort(port uint32) PeerOption {
	return func(p *Peer) {
		p.SWIMPort = port
	}
}

// PeerOptionRaftPort ...
func PeerOptionRaftPort(port uint32) PeerOption {
	return func(p *Peer) {
		p.RaftPort = port
	}
}

// PeerOptionStatus ...
func PeerOptionStatus(c Peer_State) PeerOption {
	return func(p *Peer) {
		p.Status = c
	}
}

// NewPeer ...
func NewPeer(id string, opts ...PeerOption) Peer {
	p := Peer{
		Name:     id,
		Ip:       systemx.HostIP(systemx.HostnameOrLocalhost()).String(),
		RPCPort:  2000,
		SWIMPort: 2001,
		RaftPort: 2002,
		Status:   Peer_Ready,
	}

	return NewPeerFromTemplate(p, opts...)
}

// NewPeerFromTemplate ...
func NewPeerFromTemplate(p Peer, opts ...PeerOption) Peer {
	for _, opt := range opts {
		opt(&p)
	}

	return p
}

// RPCTCPListener ...
func RPCTCPListener(t Peer) (net.Listener, error) {
	return net.Listen("tcp", net.JoinHostPort(t.Ip, strconv.Itoa(int(t.RPCPort))))
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
