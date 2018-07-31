package agent

import (
	"log"
	"time"

	"github.com/davecgh/go-spew/spew"
	"github.com/pkg/errors"
	"google.golang.org/grpc"
)

type dialer interface {
	Dial(p Peer) (Client, error)
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

	for _, q := range PtrToPeers(cinfo.Quorum...) {
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

// DefaultDialerOptions sets reasonable defaults for dialing the agent.
func DefaultDialerOptions(options ...grpc.DialOption) (results []grpc.DialOption) {
	results = make([]grpc.DialOption, 0, len(options)+1)
	results = append(
		results,
		grpc.WithBackoffMaxDelay(5*time.Second),
	)

	return append(results, options...)
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

// NewQuorumDialer creates a new dialer that connects to a member of the quorum.
func NewQuorumDialer(d dialer) QuorumDialer {
	return QuorumDialer{
		dialer: d,
	}
}

// Dialer interface for connecting to a given peer.
type Dialer struct {
	options []grpc.DialOption
}

// Dial connects to the provided peer.
func (t Dialer) Dial(p Peer) (zeroc Client, err error) {
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
func (t ProxyQuorumDialer) Dial(p Peer) (conn Client, err error) {
	if conn, err = t.d.Dial(p); err != nil {
		return conn, err
	}
	defer conn.Close()

	return ProxyDialQuorum(conn, t.d)
}

// QuorumDialer connects to a member of the quorum.
type QuorumDialer struct {
	dialer dialer
}

// Dial connects to a member of the quorum based on the cluster.
func (t QuorumDialer) Dial(c cluster) (client Client, err error) {
	for _, p := range c.Quorum() {
		log.Println("dialing", spew.Sdump(p))
		if client, err = t.dialer.Dial(p); err == nil {
			break
		}
		log.Println("failed to connect to peer", p.Name, p.Ip)
	}

	return client, errors.WithMessage(err, "failed to connect to a member of the quorum")
}
