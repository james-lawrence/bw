package agent

import (
	"fmt"
	"log"
	"net"
	"strconv"

	"github.com/golang/protobuf/proto"
	"github.com/hashicorp/memberlist"
	"github.com/james-lawrence/bw/x/systemx"
	"github.com/pkg/errors"
)

// Default ports for the agent
const (
	DefaultPortRPC     = 2000
	DefaultPortSWIM    = 2001
	DefaultPortRaft    = 2002
	DefaultPortTorrent = 2003
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

// PeerOptionName ...
func PeerOptionName(n string) PeerOption {
	return func(p *Peer) {
		p.Name = n
	}
}

// NewPeer ...
func NewPeer(id string, opts ...PeerOption) Peer {
	hn := systemx.HostnameOrLocalhost()
	p := Peer{
		Name:        id,
		Ip:          systemx.HostIP(hn).String(),
		RPCPort:     DefaultPortRPC,
		SWIMPort:    DefaultPortSWIM,
		RaftPort:    DefaultPortRaft,
		TorrentPort: DefaultPortTorrent,
		Status:      Peer_Node,
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

// NodesToPeers ...
func NodesToPeers(nodes ...*memberlist.Node) []Peer {
	peers := make([]Peer, 0, len(nodes))
	for _, n := range nodes {
		var (
			err  error
			peer Peer
		)

		if peer, err = NodeToPeer(n); err != nil {
			log.Println("skipping node", n.Name, "invalid metadata", err)
		}

		peers = append(peers, peer)
	}

	return peers
}

// PeerToNode - partial implementation, doesn't convert metadata.
// do not use, except for tests.
func PeerToNode(p Peer) memberlist.Node {
	return memberlist.Node{
		Name: p.Name,
		Addr: net.ParseIP(p.Ip),
	}
}

// NodeToPeer converts a node to a peer
func NodeToPeer(n *memberlist.Node) (_zerop Peer, err error) {
	var (
		m PeerMetadata
	)

	if err = proto.Unmarshal(n.Meta, &m); err != nil {
		return _zerop, errors.WithStack(err)
	}

	return Peer{
		Status:      Peer_State(m.Status),
		Name:        n.Name,
		Ip:          n.Addr.String(),
		RPCPort:     m.RPCPort,
		SWIMPort:    m.SWIMPort,
		RaftPort:    m.RaftPort,
		TorrentPort: m.TorrentPort,
	}, nil
}

func MustPeer(p Peer, err error) Peer {
	if err != nil {
		panic(err)
	}

	return p
}
