package clustering

import (
	"io"
	"log"

	"github.com/hashicorp/memberlist"
	"github.com/pkg/errors"
)

type printingFilter struct{}

func (t printingFilter) NotifyAlive(peer *memberlist.Node) error {
	log.Println("Alive:", peer.Name, peer.Addr.String(), int(peer.Port))
	return nil
}

// PrintingEventDelegate prints out events to the standard logger.
type PrintingEventDelegate struct{}

// NotifyJoin is invoked when a node is detected to have joined.
// The Node argument must not be modified.
func (t PrintingEventDelegate) NotifyJoin(peer *memberlist.Node) {
	log.Println("Join:", peer.Name, peer.Addr.String(), int(peer.Port))
}

// NotifyLeave is invoked when a node is detected to have left.
// The Node argument must not be modified.
func (t PrintingEventDelegate) NotifyLeave(peer *memberlist.Node) {
	log.Println("Leave:", peer.Name, peer.Addr.String(), int(peer.Port))
}

// NotifyUpdate is invoked when a node is detected to have
// updated, usually involving the meta data. The Node argument
// must not be modified.
func (t PrintingEventDelegate) NotifyUpdate(peer *memberlist.Node) {
	log.Println("Update:", peer.Name, peer.Addr.String(), int(peer.Port))
}

// Option interface for specifying a cluster option.
type Option func(*Options)

// OptionNodeID specify the name of the node within the cluster.
func OptionNodeID(nodeID string) Option {
	return func(opts *Options) {
		opts.Config.Name = nodeID
	}
}

// OptionBindAddress specify the address to bind the cluster to.
func OptionBindAddress(addr string) Option {
	return func(opts *Options) {
		opts.Config.BindAddr = addr
	}
}

// OptionBindPort specify the port for the cluster to bind to.
func OptionBindPort(port int) Option {
	return func(opts *Options) {
		opts.Config.BindPort = port
	}
}

// OptionAdvertiseAddress specify the address to advertise.
func OptionAdvertiseAddress(addr string) Option {
	return func(opts *Options) {
		opts.Config.AdvertiseAddr = addr
	}
}

// OptionAdvertisePort specify the port to advertise.
func OptionAdvertisePort(port int) Option {
	return func(opts *Options) {
		opts.Config.AdvertisePort = port
	}
}

// OptionLogger sets the logger for the memberlist.
func OptionLogger(l io.Writer) Option {
	return func(opts *Options) {
		opts.Config.LogOutput = l
	}
}

// OptionEventDelegate set the event delegate for the cluster.
func OptionEventDelegate(d memberlist.EventDelegate) Option {
	return func(opts *Options) {
		opts.Config.Events = d
	}
}

// OptionAliveDelegate set the event delegate for the cluster.
func OptionAliveDelegate(d memberlist.AliveDelegate) Option {
	return func(opts *Options) {
		log.Printf("Adding Alive Delegate: %T\n", d)
		opts.Config.Alive = d
	}
}

// OptionDelegate set the delegate for the cluster.
func OptionDelegate(delegate memberlist.Delegate) Option {
	return func(opts *Options) {
		opts.Config.Delegate = delegate
	}
}

// OptionSecret set the secret for the cluster to encrypt communications.
func OptionSecret(s []byte) Option {
	return func(opts *Options) {
		opts.Config.SecretKey = s
	}
}

// NewOptions build default cluster options for the given address:port
// combination.
func NewOptions(opts ...Option) Options {
	options := Options{
		Config: memberlist.DefaultWANConfig(),
	}

	options.Config.Events = PrintingEventDelegate{}

	for _, opt := range opts {
		opt(&options)
	}

	log.Println("Name:", options.Config.Name)
	log.Println("IndirectChecks:", options.Config.IndirectChecks)
	log.Println("RetransmitMult:", options.Config.RetransmitMult)
	log.Println("SuspicionMult:", options.Config.SuspicionMult)
	log.Println("GossipNodes:", options.Config.GossipNodes)
	log.Println("GossipInterval:", options.Config.GossipInterval)
	log.Println("disable tcp pings:", options.Config.DisableTcpPings)
	log.Println("Advertise:", options.Config.AdvertiseAddr, options.Config.AdvertisePort)
	log.Println("Bind:", options.Config.BindAddr, options.Config.BindPort)
	log.Println("TCPTimeout:", options.Config.TCPTimeout)
	log.Println("Compression:", options.Config.EnableCompression)
	log.Printf("Alive Delegate: %T\n", options.Config.Alive)

	return options
}

// Options holds the options for the cluster.
type Options struct {
	*memberlist.Config
}

// NewCluster initializes a cluster based on the options and optionally bootstraps
// the node from the provided addresses.
func (t Options) NewCluster() (Cluster, error) {
	var (
		err     error
		members *memberlist.Memberlist
		c       Cluster
	)

	if members, err = memberlist.Create(t.Config); err != nil {
		return c, errors.Wrap(err, "failed to create cluster")
	}

	return Cluster{
		list: members,
	}, nil
}
