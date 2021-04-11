package clusteringtestutil

import (
	"fmt"
	"log"
	"net"
	"sync"
	"time"

	"github.com/james-lawrence/bw"
	"github.com/james-lawrence/bw/clustering"
	"github.com/james-lawrence/bw/internal/x/stringsx"
	"github.com/pkg/errors"

	"github.com/hashicorp/memberlist"
)

var (
	incr uint8
	m    sync.Mutex
)

func nextPeer() string {
	m.Lock()
	defer m.Unlock()
	incr++
	return fmt.Sprintf("127.0.0.%d", incr)
}

func defaultLocalConfig() *memberlist.Config {
	conf := memberlist.DefaultLANConfig()
	conf.BindAddr = nextPeer()
	conf.BindPort = 0
	conf.TCPTimeout = time.Second
	conf.IndirectChecks = 1
	conf.RetransmitMult = 2
	conf.SuspicionMult = 3
	conf.PushPullInterval = 2 * time.Second
	conf.ProbeTimeout = 200 * time.Millisecond
	conf.ProbeInterval = time.Second
	conf.GossipInterval = 100 * time.Millisecond
	conf.GossipToTheDeadTime = 2 * time.Second
	return conf
}

// NewPeerFromConfig ...
func NewPeerFromConfig(config *memberlist.Config, options ...clustering.Option) (c clustering.Memberlist, err error) {
	// The mock network cannot be shutdown cleanly, so ignore it, even though it would be ideal
	// for this use case.
	// transport := network.NewTransport()
	transport, err := memberlist.NewNetTransport(&memberlist.NetTransportConfig{
		BindAddrs: []string{stringsx.DefaultIfBlank(config.BindAddr, "127.0.0.1")},
		BindPort:  config.BindPort,
	})
	if err != nil {
		return c, errors.WithMessage(err, "new cluster failed")
	}
	defaultOpts := []clustering.Option{
		clustering.OptionNodeID(bw.MustGenerateID().String()),
		clustering.OptionTransport(transport),
		clustering.OptionBindPort(transport.GetAutoBindPort()),
	}

	options = append(defaultOpts, options...)

	if c, err = clustering.NewOptionsFromConfig(config, options...).NewCluster(); err != nil {
		return c, errors.WithMessage(err, "new cluster failed")
	}

	return c, err
}

// NewPeer ...
func NewPeer(network *memberlist.MockNetwork, options ...clustering.Option) (c clustering.Memberlist, err error) {
	return NewPeerFromConfig(defaultLocalConfig(), options...)
}

// NewCluster ...
func NewCluster(n int, options ...clustering.Option) (network memberlist.MockNetwork, out []clustering.Memberlist, err error) {
	for i := 0; i < n; i++ {
		var (
			c clustering.Memberlist
		)

		if c, err = NewPeer(&network, options...); err != nil {
			return network, out, errors.WithMessage(err, "new cluster failed")
		}

		if _, err = Connect(c, out...); err != nil {
			ShutdownCluster(out...)
			ShutdownCluster(c)
			return network, out, errors.WithMessage(err, "failed to connect")
		}

		out = append(out, c)
	}

	return network, out, nil
}

// Connect the local node to the provided peers.
func Connect(local clustering.Memberlist, peers ...clustering.Memberlist) (int, error) {
	log.Println("connecting to", len(peers))
	addrs := make([]string, 0, len(peers))
	for _, p := range peers {
		addrs = append(addrs, p.LocalNode().Address())
	}

	return local.Join(addrs...)
}

// ShutdownCluster ...
func ShutdownCluster(nodes ...clustering.Memberlist) (err error) {
	for _, n := range nodes {
		if cause := n.Shutdown(); cause != nil && err == nil {
			err = cause
		}
	}

	return err
}

// NewNodes generates up to 254 peers with IPs
// between 127.0.0.1 and 127.0.0.n
func NewNodes(n int) []*memberlist.Node {
	if n >= 255 {
		panic("only supports generating a cluster up to 254 nodes")
	}

	peers := make([]*memberlist.Node, 0, n)
	for i := 0; i < n; i++ {
		peers = append(peers, NewNode(fmt.Sprintf("node-%d", i+1), net.ParseIP(fmt.Sprintf("127.0.0.%d", i+1))))
	}

	return peers
}

// NewNode creates a peer with the given name, and ip.
func NewNode(name string, ip net.IP) *memberlist.Node {
	return &memberlist.Node{
		Name: name,
		Addr: ip,
	}
}

// NewNodeFromAddress see NewNode
func NewNodeFromAddress(name, ip string) *memberlist.Node {
	return NewNode(name, net.ParseIP(ip))
}

// NewMock generates a new mock cluster with n peers.
func NewMock(n int) clustering.Mock {
	peers := NewNodes(n)
	return clustering.NewMock(peers[0], peers[1:]...)
}
