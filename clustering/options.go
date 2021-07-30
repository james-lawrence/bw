package clustering

import (
	"io"
	"log"
	"time"

	"github.com/hashicorp/memberlist"
	"github.com/pkg/errors"
)

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

// OptionLogger sets the logger for the memberlist.
func OptionLogger(l *log.Logger) Option {
	return func(opts *Options) {
		opts.Config.Logger = l
	}
}

// OptionLogOutput sets the logger for the memberlist.
func OptionLogOutput(l io.Writer) Option {
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

// OptionKeyring set the keyring for the cluster to encrypt communications.
func OptionKeyring(r *memberlist.Keyring) Option {
	return func(opts *Options) {
		opts.Config.Keyring = r
	}
}

// OptionTransport provide the transport for the node.
func OptionTransport(transport memberlist.Transport) Option {
	return func(opts *Options) {
		opts.Config.Transport = transport
	}
}

// NewOptionsFromConfig ...
func NewOptionsFromConfig(c *memberlist.Config, options ...Option) Options {
	opt := Options{
		Config: c,
	}

	OptionEventDelegate(PrintingEventDelegate{})(&opt)

	for _, option := range options {
		option(&opt)
	}

	// log.Println("Name:", opt.Config.Name)
	// log.Println("IndirectChecks:", opt.Config.IndirectChecks)
	// log.Println("RetransmitMult:", opt.Config.RetransmitMult)
	// log.Println("SuspicionMult:", opt.Config.SuspicionMult)
	// log.Println("GossipNodes:", opt.Config.GossipNodes)
	// log.Println("GossipInterval:", opt.Config.GossipInterval)
	// log.Println("disable tcp pings:", opt.Config.DisableTcpPings)
	// log.Println("Advertise:", opt.Config.AdvertiseAddr, opt.Config.AdvertisePort)
	// log.Println("Bind:", opt.Config.BindAddr, opt.Config.BindPort)
	// log.Println("TCPTimeout:", opt.Config.TCPTimeout)
	// log.Println("Compression:", opt.Config.EnableCompression)
	// log.Printf("Alive Delegate: %T\n", opt.Config.Alive)

	return opt
}

// NewOptions build default cluster options.
func NewOptions(options ...Option) Options {
	c := memberlist.DefaultWANConfig()
	c.TCPTimeout = 5 * time.Second
	c.SuspicionMult = 8
	c.GossipInterval = 2 * time.Second
	c.GossipToTheDeadTime = 240 * time.Second

	return NewOptionsFromConfig(c, options...)
}

// Options holds the options for the cluster.
type Options struct {
	*memberlist.Config
}

// NewCluster initializes a cluster based on the options and optionally bootstraps
// the node from the provided addresses.
func (t Options) NewCluster() (Memberlist, error) {
	return newCluster(t.Config)
}

// NewCluster ...
func NewCluster(options ...Option) (Memberlist, error) {
	return newCluster(NewOptions(options...).Config)
}

func newCluster(config *memberlist.Config) (Memberlist, error) {
	var (
		err     error
		members *memberlist.Memberlist
		c       Memberlist
	)

	if members, err = memberlist.Create(config); err != nil {
		return c, errors.WithStack(err)
	}

	c = Memberlist{
		config: config,
		list:   members,
	}

	return c, nil
}
