package agent

import (
	"log"
	"math/rand"

	"github.com/hashicorp/memberlist"
	"github.com/pkg/errors"
	"google.golang.org/grpc"
)

type rendezvous interface {
	GetN(int, []byte) []*memberlist.Node
}

type dialer interface {
	Dial(p *Peer) (Client, error)
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
func QuorumPeers(c rendezvous) []*Peer {
	return Shuffle(NodesToPeers(QuorumNodes(c)...))
}

// QuorumNodes return the quorum nodes.
func QuorumNodes(c rendezvous) []*memberlist.Node {
	return c.GetN(QuorumDefault, []byte(QuorumKey))
}

// LargeQuorum returns 2x the nodes required to achieve quorum.
func LargeQuorum(c rendezvous) []*memberlist.Node {
	return c.GetN(2*QuorumDefault, []byte(QuorumKey))
}

// SynchronizationPeers based on the provided k generate the peers to synchronize with
func SynchronizationPeers(k []byte, c rendezvous) []*Peer {
	return Shuffle(NodesToPeers(c.GetN(2*QuorumDefault, k)...))
}

func shuffleQuorum(q []*Peer) []*Peer {
	rand.Shuffle(len(q), func(i int, j int) {
		q[i], q[j] = q[j], q[i]
	})
	return q
}

// AddressProxyDialQuorum connects to a quorum peer using any agent for bootstrapping.
func AddressProxyDialQuorum(proxy string, options ...grpc.DialOption) (conn Client, err error) {
	if conn, err = Dial(proxy, options...); err != nil {
		return conn, err
	}
	defer conn.Close()

	return ProxyDialQuorum(conn, NewDialer(options...))
}

// ProxyDialQuorum connects to a quorum peer using any agent for bootstrapping.
func ProxyDialQuorum(c Client, d dialer) (conn Client, err error) {
	var (
		cinfo ConnectResponse
	)

	if cinfo, err = c.Connect(); err != nil {
		return conn, err
	}

	for _, q := range shuffleQuorum(cinfo.Quorum) {
		if conn, err = d.Dial(q); err != nil {
			log.Println("failed to dial", RPCAddress(q), err)
			continue
		}
		return conn, nil
	}

	return conn, errors.New("failed to bootstrap from the provided peer")
}

// Dial connects to a node at the given address.
func Dial(address string, options ...grpc.DialOption) (_ignored Conn, err error) {
	var (
		conn *grpc.ClientConn
	)

	if conn, err = grpc.Dial(address, options...); err != nil {
		return _ignored, errors.Wrap(err, "failed to connect to peer")
	}

	return Conn{conn: conn}, nil
}

// NewDialer creates a new dialer from the provided options
func NewDialer(options ...grpc.DialOption) Dialer {
	return Dialer{
		options: options,
	}
}

// NewProxyDialer creates a new dialer that connects to a member of the quorum via a proxy agent.
func NewProxyDialer(d dialer) ProxyQuorumDialer {
	return ProxyQuorumDialer{
		d: d,
	}
}

// Dialer interface for connecting to a given peer.
type Dialer struct {
	options []grpc.DialOption
}

// Dial connects to the provided peer.
func (t Dialer) Dial(p *Peer) (zeroc Client, err error) {
	var (
		addr string
	)

	if addr = RPCAddress(p); addr == "" {
		return zeroc, errors.Errorf("failed to determine address of peer: %s", p.Name)
	}

	return Dial(addr, t.options...)
}

// ProxyQuorumDialer connects to the quorum using an agent as a proxy.
type ProxyQuorumDialer struct {
	d dialer
}

// Dial a member of quorum using the provided peer as the proxy.
func (t ProxyQuorumDialer) Dial(p *Peer) (conn Client, err error) {
	if conn, err = t.d.Dial(p); err != nil {
		return conn, err
	}
	defer conn.Close()

	return ProxyDialQuorum(conn, t.d)
}
