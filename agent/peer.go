package agent

import (
	"encoding/hex"
	"fmt"
	"log"
	"net"
	"net/netip"
	"net/url"
	"strconv"

	"github.com/gofrs/uuid/v5"
	"github.com/hashicorp/memberlist"
	"github.com/james-lawrence/bw"
	"github.com/james-lawrence/bw/internal/errorsx"
	"github.com/james-lawrence/bw/internal/langx"
	"github.com/james-lawrence/bw/internal/stringsx"
	"github.com/james-lawrence/bw/internal/systemx"
	"github.com/pkg/errors"
	proto "google.golang.org/protobuf/proto"
)

// P2PAddress for a peer
func P2PRawAddress(p *Peer) string {
	if p.P2PPort == 0 {
		return ""
	}

	return net.JoinHostPort(p.Ip, fmt.Sprint(p.P2PPort))
}

// URIAgent generate a muxer protocol address.
func URIAgent(address string) string {
	return fmt.Sprintf("%s://%s", bw.ProtocolAgent, address)
}

// URIDiscovery generate a muxer protocol address.
func URIDiscovery(address string) string {
	return fmt.Sprintf("%s://%s", bw.ProtocolDiscovery, address)
}

// URIPeer for a peer
func URIPeer(p *Peer, proto string) string {
	if p.P2PPort == 0 {
		return ""
	}

	return (&url.URL{
		Scheme:   proto,
		Host:     net.JoinHostPort(p.Ip, fmt.Sprint(p.P2PPort)),
		Fragment: p.Name,
	}).String()
}

// RPCAddress for a peer.
func RPCAddress(p *Peer) string {
	return stringsx.First(URIPeer(p, bw.ProtocolAgent), net.JoinHostPort(p.Ip, fmt.Sprint(p.P2PPort)))
}

func DecodePeerID(s string) string {
	// original id format sha256 formatted hex. we're going
	// to compress it to the prefix.
	if id, err := hex.DecodeString(s); err == nil {
		switch len(id) {
		case 32:
			return bw.RandomID(id[:16]).String()
		default:
			return s
		}
	}

	if id := uuid.FromStringOrNil(s); !id.IsZero() {
		return bw.RandomID(id.Bytes()).String()
	}

	return s
}

// DiscoveryAddress for a peer.
func DiscoveryAddress(p *Peer) string {
	return stringsx.First(URIPeer(p, bw.ProtocolDiscovery), net.JoinHostPort(p.Ip, fmt.Sprint(p.P2PPort)))
}

// AutocertAddress for a peer.
func AutocertAddress(p *Peer) string {
	return stringsx.First(URIPeer(p, bw.ProtocolAutocert), net.JoinHostPort(p.Ip, fmt.Sprint(p.P2PPort)))
}

// SWIMAddress for peer.
func SWIMAddress(p *Peer) string {
	return stringsx.First(URIPeer(p, bw.ProtocolSWIM), net.JoinHostPort(p.Ip, fmt.Sprint(p.P2PPort)))
}

// RaftAddress for peer.
func RaftAddress(p *Peer) string {
	return stringsx.First(P2PRawAddress(p), net.JoinHostPort(p.Ip, fmt.Sprint(p.P2PPort)))
}

// StaticPeeringStrategy ...
func StaticPeeringStrategy(peers ...*Peer) []string {
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

// PeerOptionPublicKey peers public key.
func PeerOptionPublicKey(k []byte) PeerOption {
	return func(p *Peer) {
		p.PublicKey = k
	}
}

// NewPeer ...
func NewPeer(id string, opts ...PeerOption) *Peer {
	hn := systemx.HostnameOrLocalhost()
	p := Peer{
		Name:    id,
		Ip:      systemx.HostIP(hn).String(),
		P2PPort: bw.DefaultP2PPort,
		Status:  Peer_Node,
	}

	return NewPeerFromTemplate(&p, opts...)
}

func NewPeerZero(opts ...PeerOption) *Peer {
	var zeroid [32]byte
	return NewPeerFromTemplate(
		&Peer{
			Name:   hex.EncodeToString(zeroid[:]),
			Ip:     netip.IPv4Unspecified().String(),
			Status: Peer_Client,
		},
		langx.Compose(opts...), PeerOptionStatus(Peer_Client),
	)
}

// NewPeerFromTemplate ...
func NewPeerFromTemplate(p *Peer, opts ...PeerOption) *Peer {
	for _, opt := range opts {
		opt(p)
	}

	return p
}

func NewPeerFromURI(address string) (*Peer, error) {
	uri, err := url.Parse(address)
	if err != nil {
		return nil, errorsx.Wrapf(err, "unable to parse peer address: %s", address)
	}

	port, err := strconv.Atoi(uri.Port())
	return NewPeerFromTemplate(&Peer{
		Name:    uri.Fragment,
		Ip:      uri.Hostname(),
		P2PPort: uint32(port),
	}), err
}

// NodesToPeers ...
func NodesToPeers(nodes ...*memberlist.Node) []*Peer {
	peers := make([]*Peer, 0, len(nodes))
	for _, n := range nodes {
		var (
			err  error
			peer *Peer
		)

		if peer, err = NodeToPeer(n); err != nil {
			log.Println("skipping node", n.Name, "invalid metadata", err)
			continue
		}

		peers = append(peers, peer)
	}

	return peers
}

// PeerToMetadata ...
func PeerToMetadata(p *Peer) *PeerMetadata {
	return &PeerMetadata{
		Status:  int32(p.Status),
		P2PPort: p.P2PPort,
	}
}

func EncodeMetadata(p *PeerMetadata) ([]byte, error) {
	return proto.Marshal(p)
}

// PeerToNode converts a Peer to a memberlist.Node
func PeerToNode(p *Peer) *memberlist.Node {
	meta, err := EncodeMetadata(PeerToMetadata(p))
	if err != nil {
		panic(errors.Wrap(err, "failed to marshal peer to metadata"))
	}

	return &memberlist.Node{
		Name: p.Name,
		Addr: net.ParseIP(p.Ip),
		Port: uint16(p.P2PPort),
		Meta: meta,
	}
}

func PeersToNodes(peers ...*Peer) (nodes []*memberlist.Node) {
	for _, p := range peers {
		nodes = append(nodes, PeerToNode(p))
	}
	return nodes
}

// NodeToPeer converts a node to a peer
func NodeToPeer(n *memberlist.Node) (_zerop *Peer, err error) {
	var (
		m PeerMetadata
	)

	if n == nil {
		return nil, errors.New("unable to convert nil memberlist to node")
	}

	if err = proto.Unmarshal(n.Meta, &m); err != nil {
		return nil, errors.WithStack(err)
	}

	return &Peer{
		Status:  Peer_State(m.Status),
		Name:    n.Name,
		Ip:      n.Addr.String(),
		P2PPort: m.P2PPort,
	}, nil
}

// MustPeer if err is not nil panics.
func MustPeer(p *Peer, err error) *Peer {
	if err != nil {
		panic(err)
	}

	return p
}
